package web

import (
	"net/http"
	"strings"

	"github.com/sergii/specview/internal/executionhistory"
	"github.com/sergii/specview/internal/hoststate"
)

type historyPageData struct {
	Hostname   string
	Projection executionhistory.Projection
	Repository *hoststate.Repository
}

func (s *HostServer) historyPage(w http.ResponseWriter, r *http.Request) {
	projection, err := executionhistory.NewReader(s.catalog).Build(r.Context())
	if err != nil {
		http.Error(w, "load execution history: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	var repository *hoststate.Repository
	if repositoryID := strings.TrimSpace(r.URL.Query().Get("repository")); repositoryID != "" {
		repo, ok := s.catalog.Find(repositoryID)
		if !ok {
			http.NotFound(w, r)
			return
		}
		entries := make([]executionhistory.Entry, 0, len(projection.Entries))
		for _, entry := range projection.Entries {
			if entry.RepositoryID == repo.ID {
				entries = append(entries, entry)
			}
		}
		projection.Entries = entries
		repository = &repo
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := s.tmpl.ExecuteTemplate(w, "history.html", historyPageData{
		Hostname:   s.catalog.Hostname(),
		Projection: projection,
		Repository: repository,
	}); err != nil {
		http.Error(w, "render execution history", http.StatusInternalServerError)
	}
}
