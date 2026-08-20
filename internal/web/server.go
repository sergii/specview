package web

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"hash/fnv"
	"html/template"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/sergii/specview/internal/activity"
	"github.com/sergii/specview/internal/config"
	"github.com/sergii/specview/internal/specs"
)

//go:embed templates/*
var templateFS embed.FS

var explicitSpecIDPattern = regexp.MustCompile(`(?i)^([a-z]{1,8}(?:[-_]?\d{1,5}))(?:[-_]|$)`)

type Hub struct {
	mu      sync.Mutex
	clients map[chan struct{}]struct{}
}

func NewHub() *Hub { return &Hub{clients: make(map[chan struct{}]struct{})} }
func (h *Hub) Subscribe() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() { h.mu.Lock(); delete(h.clients, ch); close(ch); h.mu.Unlock() }
}
func (h *Hub) Broadcast() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

type ProjectSource struct {
	Key      string
	Root     string
	Config   config.Config
	Store    *specs.Store
	Activity *activity.Store
}

func NewProjectSource(root string, cfg config.Config, store *specs.Store, activityStore *activity.Store) ProjectSource {
	return ProjectSource{
		Key:      projectSourceKey(root),
		Root:     filepath.Clean(root),
		Config:   cfg,
		Store:    store,
		Activity: activityStore,
	}
}

type Server struct {
	projects  []ProjectSource
	serverCfg config.Server
	hub       *Hub
	tmpl      *template.Template
}

func NewServer(root string, cfg config.Config, store *specs.Store, hub *Hub) *Server {
	return NewWorkspaceServer([]ProjectSource{NewProjectSource(root, cfg, store, nil)}, cfg.Server, hub)
}

func NewWorkspaceServer(projects []ProjectSource, serverCfg config.Server, hub *Hub) *Server {
	funcs := template.FuncMap{
		"since":      func(t time.Time) string { return since(t) },
		"specID":     specDisplayID,
		"agentLabel": activity.AgentLabel,
		"expiresAt": func(record activity.Record) string {
			return activity.ExpiresAt(record).UTC().Format(time.RFC3339Nano)
		},
	}
	tmpl := template.Must(template.New("index.html").Funcs(funcs).ParseFS(templateFS, "templates/*.html"))
	return &Server{projects: projects, serverCfg: serverCfg, hub: hub, tmpl: tmpl}
}

func (s *Server) SetActivityStore(store *activity.Store) {
	if len(s.projects) == 0 {
		return
	}
	s.projects[0].Activity = store
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.index)
	mux.HandleFunc("GET /spec", s.detail)
	mux.HandleFunc("GET /events", s.events)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.serverCfg.Host, s.serverCfg.Port),
		Handler: securityHeaders(mux),
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() { errCh <- server.ListenAndServe() }()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

type boardSpec struct {
	specs.Spec
	ProjectKey string
	Activity   []activity.Record
}

type boardData struct {
	ProjectKey, ProjectName, ProjectPath, ProjectDisplayPath, SpecsPath string
	New, InProgress, Done, Invalid                                      []boardSpec
	Total                                                               int
}

func projectName(source ProjectSource) string {
	if name := strings.TrimSpace(source.Config.Project.Name); name != "" {
		return name
	}
	return filepath.Base(source.Root)
}

func compactProjectPath(root string) string {
	clean := filepath.Clean(root)
	base := filepath.Base(clean)
	parentPath := filepath.Dir(clean)
	parent := filepath.Base(parentPath)
	if parent == "." || parent == string(filepath.Separator) || parent == base {
		return filepath.ToSlash(base)
	}
	return filepath.ToSlash(filepath.Join(parent, base))
}

func projectSourceKey(root string) string {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(filepath.ToSlash(filepath.Clean(root))))
	return fmt.Sprintf("p-%08x", hash.Sum32())
}

func present(source ProjectSource, item specs.Spec, now time.Time) boardSpec {
	view := boardSpec{Spec: item, ProjectKey: source.Key}
	if source.Activity == nil {
		return view
	}
	projectRelativePath := filepath.ToSlash(filepath.Join(source.Config.Specs.Path, item.Path))
	view.Activity = source.Activity.ActiveFor(projectRelativePath, now)
	return view
}

func buildBoardData(source ProjectSource, now time.Time) boardData {
	items := source.Store.All()
	data := boardData{
		ProjectKey:         source.Key,
		ProjectName:        projectName(source),
		ProjectPath:        filepath.ToSlash(filepath.Clean(source.Root)),
		ProjectDisplayPath: compactProjectPath(source.Root),
		SpecsPath:          source.Config.Specs.Path,
		Total:              len(items),
	}
	for _, item := range items {
		view := present(source, item, now)
		if item.Error != "" {
			data.Invalid = append(data.Invalid, view)
			continue
		}
		switch item.Status {
		case specs.StatusNew:
			data.New = append(data.New, view)
		case specs.StatusInProgress:
			data.InProgress = append(data.InProgress, view)
		case specs.StatusDone:
			data.Done = append(data.Done, view)
		}
	}
	return data
}

func (s *Server) index(w http.ResponseWriter, _ *http.Request) {
	if len(s.projects) == 0 {
		http.Error(w, "no projects configured", http.StatusInternalServerError)
		return
	}
	data := buildBoardData(s.projects[0], time.Now())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "render dashboard", http.StatusInternalServerError)
	}
}

func (s *Server) projectForRequest(r *http.Request) (ProjectSource, bool) {
	if len(s.projects) == 0 {
		return ProjectSource{}, false
	}
	key := r.URL.Query().Get("project")
	if key == "" {
		return s.projects[0], true
	}
	for _, source := range s.projects {
		if source.Key == key {
			return source, true
		}
	}
	return ProjectSource{}, false
}

func (s *Server) detail(w http.ResponseWriter, r *http.Request) {
	source, ok := s.projectForRequest(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	path := r.URL.Query().Get("path")
	item, ok := source.Store.Find(path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "detail.html", struct {
		ProjectName string
		Spec        specs.Spec
	}{projectName(source), item}); err != nil {
		http.Error(w, "render specification", http.StatusInternalServerError)
	}
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	ch, unsubscribe := s.hub.Subscribe()
	defer unsubscribe()
	_, _ = fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()
	keepAlive := time.NewTicker(20 * time.Second)
	defer keepAlive.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case _, ok := <-ch:
			if !ok {
				return
			}
			_, _ = fmt.Fprint(w, "event: changed\ndata: refresh\n\n")
			flusher.Flush()
		case <-keepAlive.C:
			_, _ = fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; connect-src 'self'")
		next.ServeHTTP(w, r)
	})
}

func specDisplayID(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if match := explicitSpecIDPattern.FindStringSubmatch(base); len(match) == 2 {
		return strings.ToUpper(strings.ReplaceAll(match[1], "_", "-"))
	}

	hash := fnv.New32a()
	_, _ = hash.Write([]byte(filepath.ToSlash(path)))
	value := hash.Sum32()
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	id := [5]byte{}
	for i := len(id) - 1; i >= 0; i-- {
		id[i] = alphabet[value%36]
		value /= 36
	}
	return string(id[:])
}

func since(t time.Time) string {
	d := time.Since(t)
	if d < 0 {
		return "just now"
	}
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}
