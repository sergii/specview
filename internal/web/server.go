package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"html/template"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
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

const orphanedAfter = 15 * time.Minute

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
		"since":        func(t time.Time) string { return since(t) },
		"specID":       specDisplayID,
		"markdownBody": renderMarkdownBody,
		"agentLabel":   activity.AgentLabel,
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
	mux.HandleFunc("GET /graph", s.graphPage)
	mux.HandleFunc("GET /api/graph", s.graph)
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
	ProjectKey     string
	Activity       []activity.Record
	Orphaned       bool
	AgentCollision bool
	CodeCollisions []string
}

type boardData struct {
	ProjectKey, ProjectName, ProjectPath, ProjectDisplayPath, SpecsPath string
	New, InProgress, Done, Invalid                                      []boardSpec
	Total                                                               int
	ActiveSessions                                                      int
	OrphanedCount                                                       int
	CollisionCount                                                      int
}

type workspaceData struct {
	boardData
	Projects       []boardData
	WorkspaceTotal int
	Multi          bool
}

type graphData struct {
	Nodes []graphNode `json:"nodes"`
	Edges []graphEdge `json:"edges"`
}

type graphNode struct {
	ID           string       `json:"id"`
	ProjectKey   string       `json:"project_key"`
	Project      string       `json:"project"`
	SpecID       string       `json:"spec_id"`
	Path         string       `json:"path"`
	Title        string       `json:"title"`
	Status       specs.Status `json:"status"`
	Agents       []string     `json:"agents,omitempty"`
	Orphaned     bool         `json:"orphaned,omitempty"`
	Collision    bool         `json:"collision,omitempty"`
	ModifiedUnix int64        `json:"modified_unix"`
}

type graphEdge struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Type    string `json:"type"`
	Target  string `json:"target,omitempty"`
	Missing bool   `json:"missing,omitempty"`
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
	if source.Activity != nil {
		projectRelativePath := filepath.ToSlash(filepath.Join(source.Config.Specs.Path, item.Path))
		view.Activity = source.Activity.ActiveFor(projectRelativePath, now)
	}
	view.Orphaned = item.Status == specs.StatusInProgress && len(view.Activity) == 0 && now.Sub(item.ModifiedAt) >= orphanedAfter
	view.AgentCollision = len(view.Activity) > 1
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
	annotateBoardSignals(&data)
	return data
}

func annotateBoardSignals(data *boardData) {
	groups := []*[]boardSpec{&data.New, &data.InProgress, &data.Done, &data.Invalid}
	fileOwners := make(map[string]map[string]struct{})
	activeSessions := make(map[string]struct{})
	collisionSpecs := make(map[string]struct{})

	for _, group := range groups {
		for i := range *group {
			item := &(*group)[i]
			if item.Orphaned {
				data.OrphanedCount++
			}
			if item.AgentCollision {
				collisionSpecs[item.Path] = struct{}{}
			}
			for _, record := range item.Activity {
				activeSessions[record.SessionID] = struct{}{}
				for _, file := range record.Files {
					owners := fileOwners[file]
					if owners == nil {
						owners = make(map[string]struct{})
						fileOwners[file] = owners
					}
					owners[item.Path] = struct{}{}
				}
			}
		}
	}
	data.ActiveSessions = len(activeSessions)

	for _, group := range groups {
		for i := range *group {
			item := &(*group)[i]
			collisions := make([]string, 0)
			seen := make(map[string]struct{})
			for _, record := range item.Activity {
				for _, file := range record.Files {
					if len(fileOwners[file]) < 2 {
						continue
					}
					if _, ok := seen[file]; ok {
						continue
					}
					seen[file] = struct{}{}
					collisions = append(collisions, file)
				}
			}
			sort.Strings(collisions)
			item.CodeCollisions = collisions
			if len(collisions) > 0 {
				collisionSpecs[item.Path] = struct{}{}
			}
		}
	}
	data.CollisionCount = len(collisionSpecs)
}

func boardSpecs(data boardData) []boardSpec {
	items := make([]boardSpec, 0, data.Total)
	items = append(items, data.New...)
	items = append(items, data.InProgress...)
	items = append(items, data.Done...)
	items = append(items, data.Invalid...)
	return items
}

func (s *Server) graphData(now time.Time) graphData {
	result := graphData{}
	seenEdges := make(map[string]struct{})

	for _, source := range s.projects {
		board := buildBoardData(source, now)
		views := boardSpecs(board)
		refs := make(map[string]string, len(views)*3)
		for _, view := range views {
			nodeID := graphNodeID(source.Key, view.Path)
			refs[view.Path] = nodeID
			refs[filepath.ToSlash(filepath.Join(source.Config.Specs.Path, view.Path))] = nodeID
			refs[strings.ToUpper(specDisplayID(view.Path))] = nodeID

			agents := make([]string, 0, len(view.Activity))
			seenAgents := make(map[string]struct{})
			for _, record := range view.Activity {
				label := activity.AgentLabel(record)
				if _, ok := seenAgents[label]; ok {
					continue
				}
				seenAgents[label] = struct{}{}
				agents = append(agents, label)
			}
			sort.Strings(agents)
			result.Nodes = append(result.Nodes, graphNode{
				ID:           nodeID,
				ProjectKey:   source.Key,
				Project:      compactProjectPath(source.Root),
				SpecID:       specDisplayID(view.Path),
				Path:         view.Path,
				Title:        view.Title,
				Status:       view.Status,
				Agents:       agents,
				Orphaned:     view.Orphaned,
				Collision:    view.AgentCollision || len(view.CodeCollisions) > 0,
				ModifiedUnix: view.ModifiedAt.Unix(),
			})
		}

		for _, view := range views {
			fromID := graphNodeID(source.Key, view.Path)
			for _, target := range view.DependsOn {
				toID, ok := resolveRelation(target, refs)
				edge := graphEdge{From: toID, To: fromID, Type: "depends_on", Target: target, Missing: !ok}
				if !ok {
					edge.From = ""
				}
				appendGraphEdge(&result, seenEdges, edge)
			}
			for _, target := range view.Blocks {
				toID, ok := resolveRelation(target, refs)
				edge := graphEdge{From: fromID, To: toID, Type: "blocks", Target: target, Missing: !ok}
				if !ok {
					edge.To = ""
				}
				appendGraphEdge(&result, seenEdges, edge)
			}
		}
	}

	sort.Slice(result.Nodes, func(i, j int) bool {
		if result.Nodes[i].Project != result.Nodes[j].Project {
			return result.Nodes[i].Project < result.Nodes[j].Project
		}
		return result.Nodes[i].Path < result.Nodes[j].Path
	})
	return result
}

func graphNodeID(projectKey, path string) string {
	return projectKey + "::" + filepath.ToSlash(path)
}

func resolveRelation(target string, refs map[string]string) (string, bool) {
	trimmed := strings.TrimSpace(target)
	candidates := []string{
		trimmed,
		filepath.ToSlash(filepath.Clean(trimmed)),
		strings.ToUpper(trimmed),
	}
	for _, candidate := range candidates {
		if id, ok := refs[candidate]; ok {
			return id, true
		}
	}
	return "", false
}

func appendGraphEdge(result *graphData, seen map[string]struct{}, edge graphEdge) {
	key := strings.Join([]string{edge.From, edge.To, edge.Type, edge.Target}, "\x1f")
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	result.Edges = append(result.Edges, edge)
}

func (s *Server) index(w http.ResponseWriter, _ *http.Request) {
	if len(s.projects) == 0 {
		http.Error(w, "no projects configured", http.StatusInternalServerError)
		return
	}
	now := time.Now()
	boards := make([]boardData, 0, len(s.projects))
	workspaceTotal := 0
	for _, source := range s.projects {
		board := buildBoardData(source, now)
		boards = append(boards, board)
		workspaceTotal += board.Total
	}
	data := workspaceData{
		boardData:      boards[0],
		Projects:       boards,
		WorkspaceTotal: workspaceTotal,
		Multi:          len(boards) > 1,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "render dashboard", http.StatusInternalServerError)
	}
}

func (s *Server) graphPage(w http.ResponseWriter, r *http.Request) {
	mode := "2d"
	if r.URL.Query().Get("mode") == "3d" {
		mode = "3d"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "graph.html", struct{ Mode string }{Mode: mode}); err != nil {
		http.Error(w, "render graph", http.StatusInternalServerError)
	}
}

func (s *Server) graph(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(s.graphData(time.Now())); err != nil {
		http.Error(w, "render graph", http.StatusInternalServerError)
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
