package web

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/sergii/specview/internal/federationruntime"
)

// FederationReader provides the existing deterministic multi-Host projection.
// The web layer is an adapter and must not own federation semantics.
type FederationReader interface {
	Build(context.Context) (federationruntime.Projection, error)
}

type federationPageData struct {
	Hostname   string
	Projection federationruntime.Projection
}

func (s *HostServer) federationPage(reader FederationReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if reader == nil {
			http.Error(w, "federation projection unavailable", http.StatusServiceUnavailable)
			return
		}
		projection, err := reader.Build(r.Context())
		if err != nil {
			http.Error(w, "load federation projection: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if err := s.tmpl.ExecuteTemplate(w, "federation.html", federationPageData{
			Hostname:   s.catalog.Hostname(),
			Projection: projection,
		}); err != nil {
			http.Error(w, "render federation", http.StatusInternalServerError)
		}
	}
}

// ListenAndServeWithFederation exposes the normal Host UI plus the federation
// read surface. It intentionally shares the Host listener and security headers;
// H28 does not create another network authority or listener.
func (s *HostServer) ListenAndServeWithFederation(ctx context.Context, federation FederationReader) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.index)
	mux.HandleFunc("GET /fragments/host", s.hostFragment)
	mux.HandleFunc("GET /project", s.project)
	mux.HandleFunc("GET /fragments/project", s.projectFragment)
	mux.HandleFunc("GET /project/spec", s.projectSpec)
	mux.HandleFunc("GET /federation", s.federationPage(federation))
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
