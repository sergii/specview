package web

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sergii/specview/internal/config"
	"github.com/sergii/specview/internal/specs"
)

//go:embed templates/*
var templateFS embed.FS

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

type Server struct {
	root  string
	cfg   config.Config
	store *specs.Store
	hub   *Hub
	tmpl  *template.Template
}

func NewServer(root string, cfg config.Config, store *specs.Store, hub *Hub) *Server {
	funcs := template.FuncMap{"since": func(t time.Time) string { return since(t) }}
	tmpl := template.Must(template.New("index.html").Funcs(funcs).ParseFS(templateFS, "templates/*.html"))
	return &Server{root: root, cfg: cfg, store: store, hub: hub, tmpl: tmpl}
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
		Addr:    fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port),
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

type boardData struct {
	ProjectName, SpecsPath         string
	Demo                           bool
	New, InProgress, Done, Invalid []specs.Spec
	Total                          int
}

func (s *Server) projectName() string {
	if name := strings.TrimSpace(s.cfg.Project.Name); name != "" {
		return name
	}
	return filepath.Base(s.root)
}
func (s *Server) index(w http.ResponseWriter, _ *http.Request) {
	items := s.store.All()
	data := boardData{ProjectName: s.projectName(), SpecsPath: s.cfg.Specs.Path, Demo: s.cfg.Project.Demo, Total: len(items)}
	for _, item := range items {
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "render dashboard", http.StatusInternalServerError)
	}
}
func (s *Server) detail(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	item, ok := s.store.Find(path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "detail.html", struct {
		ProjectName string
		Demo        bool
		Spec        specs.Spec
	}{s.projectName(), s.cfg.Project.Demo, item}); err != nil {
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
