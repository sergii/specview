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
	"strings"
	"time"

	"github.com/sergii/specview/internal/config"
	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/sourcecontrol"
	"github.com/sergii/specview/internal/specs"
)

type RepositorySearcher interface {
	SearchRepositoryIDs(context.Context, string, int) ([]string, error)
}

type HostServer struct {
	catalog          *hoststate.Catalog
	hub              *Hub
	host             string
	port             int
	tmpl             *template.Template
	executionSource  hoststate.ExecutionSource
	sourceControl    sourcecontrol.Source
	repositorySearch RepositorySearcher
}

func NewHostServer(catalog *hoststate.Catalog, hub *Hub, host string, port int, executionSources ...hoststate.ExecutionSource) *HostServer {
	var executionSource hoststate.ExecutionSource = hoststate.DefaultExecutionRegistry()
	if len(executionSources) > 0 && executionSources[0] != nil {
		executionSource = executionSources[0]
	}
	return newHostServer(catalog, hub, host, port, executionSource, sourcecontrol.DefaultService(), nil)
}

func NewHostServerWithSources(catalog *hoststate.Catalog, hub *Hub, host string, port int, executionSource hoststate.ExecutionSource, sourceControl sourcecontrol.Source, searchers ...RepositorySearcher) *HostServer {
	if executionSource == nil {
		executionSource = hoststate.DefaultExecutionRegistry()
	}
	if sourceControl == nil {
		sourceControl = sourcecontrol.DefaultService()
	}
	var searcher RepositorySearcher
	if len(searchers) > 0 {
		searcher = searchers[0]
	}
	return newHostServer(catalog, hub, host, port, executionSource, sourceControl, searcher)
}

func NewHostServerWithSearch(catalog *hoststate.Catalog, hub *Hub, host string, port int, executionSource hoststate.ExecutionSource, searcher RepositorySearcher) *HostServer {
	if executionSource == nil {
		executionSource = hoststate.DefaultExecutionRegistry()
	}
	return newHostServer(catalog, hub, host, port, executionSource, sourcecontrol.DefaultService(), searcher)
}

func newHostServer(catalog *hoststate.Catalog, hub *Hub, host string, port int, executionSource hoststate.ExecutionSource, sourceControl sourcecontrol.Source, searcher RepositorySearcher) *HostServer {
	funcs := template.FuncMap{
		"since": func(t time.Time) string { return since(t) },
		"projectItem": func(repoID string, item specs.Artifact) projectItemData {
			return projectItemData{RepoID: repoID, Item: item}
		},
	}
	tmpl := template.Must(template.New("host.html").Funcs(funcs).ParseFS(templateFS, "templates/*.html"))
	return &HostServer{
		catalog:          catalog,
		hub:              hub,
		host:             host,
		port:             port,
		tmpl:             tmpl,
		executionSource:  executionSource,
		sourceControl:    sourceControl,
		repositorySearch: searcher,
	}
}

func (s *HostServer) ListenAndServe(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.index)
	mux.HandleFunc("GET /fragments/host", s.hostFragment)
	mux.HandleFunc("GET /project", s.project)
	mux.HandleFunc("GET /fragments/project", s.projectFragment)
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
	Hostname    string
	Query       string
	Filtered    bool
	SearchError string
	Results     []hoststate.Repository
	Today       []hoststate.Repository
	Yesterday   []hoststate.Repository
	Earlier     []hoststate.Repository
	Total       int
}

func (s *HostServer) index(w http.ResponseWriter, r *http.Request) {
	data := s.loadHostData(r.Context(), r.URL.Query().Get("q"), time.Now())

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "host.html", data); err != nil {
		http.Error(w, "render host dashboard", http.StatusInternalServerError)
	}
}

func (s *HostServer) hostFragment(w http.ResponseWriter, r *http.Request) {
	data := s.loadHostData(r.Context(), r.URL.Query().Get("q"), time.Now())

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := s.tmpl.ExecuteTemplate(w, "host-results", data); err != nil {
		http.Error(w, "render host fragment", http.StatusInternalServerError)
	}
}

func (s *HostServer) loadHostData(ctx context.Context, rawQuery string, now time.Time) hostData {
	startToday := startOfDay(now)
	startYesterday := startToday.AddDate(0, 0, -1)

	query := strings.TrimSpace(rawQuery)
	data := hostData{Hostname: s.catalog.Hostname(), Query: query, Filtered: query != ""}
	repositories := s.catalog.Repositories()

	if query != "" {
		if s.repositorySearch == nil {
			data.SearchError = "Host index unavailable."
			return data
		}

		ids, err := s.repositorySearch.SearchRepositoryIDs(ctx, query, 100)
		if err != nil {
			data.SearchError = "Host index search unavailable."
			return data
		}

		matches := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			matches[id] = struct{}{}
		}
		for _, repo := range repositories {
			if _, ok := matches[repo.ID]; !ok {
				continue
			}
			data.Results = append(data.Results, repo)
		}
		data.Total = len(data.Results)
		return data
	}

	for _, repo := range repositories {
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
	return data
}

type projectItemData struct {
	RepoID string
	Item   specs.Artifact
}

type projectData struct {
	Repo              hoststate.Repository
	Execution         hoststate.RepositoryExecutionView
	SourceControl     sourcecontrol.RepositoryContext
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
	data, ok := s.projectDataForRequest(w, r)
	if !ok {
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "project.html", data); err != nil {
		http.Error(w, "render repository", http.StatusInternalServerError)
	}
}

func (s *HostServer) projectFragment(w http.ResponseWriter, r *http.Request) {
	data, ok := s.projectDataForRequest(w, r)
	if !ok {
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := s.tmpl.ExecuteTemplate(w, "project-live", data); err != nil {
		http.Error(w, "render repository fragment", http.StatusInternalServerError)
	}
}

func (s *HostServer) projectDataForRequest(w http.ResponseWriter, r *http.Request) (projectData, bool) {
	repo, ok := s.catalog.Find(r.URL.Query().Get("id"))
	if !ok {
		http.NotFound(w, r)
		return projectData{}, false
	}

	data, err := s.loadProject(r.Context(), repo)
	if err != nil {
		http.Error(w, "load repository: "+err.Error(), http.StatusInternalServerError)
		return projectData{}, false
	}
	return data, true
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

func (s *HostServer) loadProject(ctx context.Context, repo hoststate.Repository) (projectData, error) {
	data, store, err := s.projectStore(repo)
	if err != nil {
		return projectData{}, err
	}

	repositoryContext, err := s.sourceControl.Inspect(ctx, repo.Root)
	if err != nil {
		return projectData{}, err
	}
	data.SourceControl = repositoryContext
	data.Execution = repo.ExecutionViewWithGit(repositoryContext.Git, s.executionSource)

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

	scope := strings.TrimSpace(r.URL.Query().Get("scope"))
	if scope == "" {
		scope = "host"
	}
	projectID := strings.TrimSpace(r.URL.Query().Get("id"))
	fingerprint, err := s.materialFingerprint(r.Context(), scope, projectID)
	if err != nil {
		http.Error(w, "initialize live projection", http.StatusBadRequest)
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

	probe := time.NewTicker(time.Second)
	defer probe.Stop()
	keepAlive := time.NewTicker(20 * time.Second)
	defer keepAlive.Stop()

	emitIfChanged := func() {
		next, err := s.materialFingerprint(r.Context(), scope, projectID)
		if err != nil || next == fingerprint {
			return
		}
		fingerprint = next
		_, _ = fmt.Fprint(w, "event: changed\ndata: refresh\n\n")
		flusher.Flush()
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case _, ok := <-ch:
			if !ok {
				return
			}
			emitIfChanged()
		case <-probe.C:
			emitIfChanged()
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
