package web

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sergii/specview/internal/config"
	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/specs"
)

type HostServer struct {
	catalog *hoststate.Catalog
	hub     *Hub
	host    string
	port    int
	tmpl    *template.Template
}

func NewHostServer(catalog *hoststate.Catalog, hub *Hub, host string, port int) *HostServer {
	funcs := template.FuncMap{
		"since": func(t time.Time) string { return since(t) },
		"projectItem": func(repoID string, item specs.Artifact) projectItemData {
			return projectItemData{RepoID: repoID, Item: item}
		},
	}
	tmpl := template.Must(template.New("host.html").Funcs(funcs).ParseFS(templateFS, "templates/*.html"))
	return &HostServer{
		catalog: catalog,
		hub:     hub,
		host:    host,
		port:    port,
		tmpl:    tmpl,
	}
}

func (s *HostServer) ListenAndServe(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.index)
	mux.HandleFunc("GET /project", s.project)
	mux.HandleFunc("GET /project/spec", s.projectSpec)
	mux.HandleFunc("GET /events", s.events)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.host, s.port),
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

type hostData struct {
	Hostname  string
	Today     []hoststate.Repository
	Yesterday []hoststate.Repository
	Earlier   []hoststate.Repository
	Total     int
}

func (s *HostServer) index(w http.ResponseWriter, _ *http.Request) {
	now := time.Now()
	startToday := startOfDay(now)
	startYesterday := startToday.AddDate(0, 0, -1)

	data := hostData{Hostname: s.catalog.Hostname()}
	for _, repo := range s.catalog.Repositories() {
		data.Total++
		switch {
		case !repo.LastSeenAt.Before(startToday):
			data.Today = append(data.Today, repo)
		case !repo.LastSeenAt.Before(startYesterday):
			data.Yesterday = append(data.Yesterday, repo)
		default:
			data.Earlier = append(data.Earlier, repo)
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "host.html", data); err != nil {
		http.Error(w, "render host dashboard", http.StatusInternalServerError)
	}
}

type projectItemData struct {
	RepoID string
	Item   specs.Artifact
}

type projectData struct {
	Repo              hoststate.Repository
	Convention        config.Convention
	DetectionError    string
	Unsupported       bool
	New               []specs.Artifact
	InProgress        []specs.Artifact
	Done              []specs.Artifact
	Invalid           []specs.Artifact
	Total             int
	SpecificationRoot string
}

func (s *HostServer) project(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.catalog.Find(r.URL.Query().Get("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	data, err := s.loadProject(repo)
	if err != nil {
		http.Error(w, "load repository: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "project.html", data); err != nil {
		http.Error(w, "render repository", http.StatusInternalServerError)
	}
}

func (s *HostServer) projectSpec(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.catalog.Find(r.URL.Query().Get("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	data, store, err := s.projectStore(repo)
	if err != nil || store == nil {
		http.NotFound(w, r)
		return
	}
	item, ok := store.Find(r.URL.Query().Get("path"))
	if !ok || !item.IsBoardItem() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "detail.html", struct {
		ProjectName string
		Spec        specs.Spec
	}{
		ProjectName: data.Repo.Name,
		Spec:        item,
	}); err != nil {
		http.Error(w, "render specification", http.StatusInternalServerError)
	}
}

func (s *HostServer) loadProject(repo hoststate.Repository) (projectData, error) {
	data, store, err := s.projectStore(repo)
	if err != nil {
		return projectData{}, err
	}
	if store == nil {
		return data, nil
	}
	for _, item := range store.All() {
		if !item.IsBoardItem() {
			continue
		}
		data.Total++
		if item.Error != "" {
			data.Invalid = append(data.Invalid, item)
			continue
		}
		switch item.Status {
		case specs.StatusNew:
			data.New = append(data.New, item)
		case specs.StatusInProgress:
			data.InProgress = append(data.InProgress, item)
		case specs.StatusDone:
			data.Done = append(data.Done, item)
		}
	}
	return data, nil
}

func (s *HostServer) projectStore(repo hoststate.Repository) (projectData, *specs.Store, error) {
	convention, detectErr := config.DetectConvention(repo.Root)
	data := projectData{
		Repo:           repo,
		Convention:     convention,
		DetectionError: "",
	}
	if detectErr != nil {
		data.DetectionError = detectErr.Error()
		return data, nil, nil
	}
	if !convention.Recognized {
		return data, nil, nil
	}
	if !convention.Supported {
		data.Unsupported = true
		return data, nil, nil
	}

	projectRoot := repo.Root
	pattern := "*.md"
	specPath := convention.Path

	if _, err := os.Stat(filepath.Join(repo.Root, config.FileName)); err == nil {
		cfg, err := config.Load(repo.Root)
		if err != nil {
			return projectData{}, nil, err
		}
		projectRoot = cfg.ResolveProjectRoot(repo.Root)
		pattern = cfg.Specs.Pattern
		specPath = cfg.Specs.Path
		convention = config.Convention{
			Adapter:    cfg.Specs.Adapter,
			Label:      config.ConventionLabel(cfg.Specs.Adapter),
			Path:       cfg.Specs.Path,
			Recognized: true,
			Supported:  cfg.Specs.Adapter == config.AdapterSpecview || cfg.Specs.Adapter == config.AdapterGitHubSpecKit || cfg.Specs.Adapter == config.AdapterOpenSpec,
		}
		data.Convention = convention
	}

	specRoot := filepath.Join(projectRoot, specPath)
	data.SpecificationRoot = filepath.ToSlash(specPath)
	adapter, err := specs.NewAdapter(convention.Adapter, specRoot, pattern)
	if err != nil {
		return projectData{}, nil, err
	}
	store := specs.NewStoreWithAdapter(adapter)
	if err := store.Refresh(); err != nil {
		return projectData{}, nil, err
	}
	return data, store, nil
}

func (s *HostServer) events(w http.ResponseWriter, r *http.Request) {
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

func startOfDay(t time.Time) time.Time {
	local := t.In(t.Location())
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
}
