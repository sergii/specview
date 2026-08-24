package web

import (
	"net/http"
	"strings"
	"time"

	"github.com/sergii/specview/internal/evidence"
	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/projectstate"
)

type repositoryEvidencePageData struct {
	Hostname   string
	Repository hoststate.Repository
	Overview   projectstate.EvidenceOverview
	Rows       []repositoryEvidenceRow
	Error      string
}

type repositoryEvidenceRow struct {
	Record        evidence.Record
	WorkItemPath  string
	WorkItemTitle string
	ObservedAt    string
}

func (s *HostServer) repositoryEvidencePage(w http.ResponseWriter, r *http.Request) {
	repositoryID := strings.TrimSpace(r.URL.Query().Get("id"))
	repository, ok := s.catalog.Find(repositoryID)
	if !ok {
		http.NotFound(w, r)
		return
	}

	data := repositoryEvidencePageData{
		Hostname:   s.catalog.Hostname(),
		Repository: repository,
	}
	project, err := projectstate.Resolve(repository.Root)
	if err != nil {
		data.Error = err.Error()
		s.renderRepositoryEvidence(w, data)
		return
	}
	overview, err := project.EvidenceOverview()
	if err != nil {
		data.Error = err.Error()
		s.renderRepositoryEvidence(w, data)
		return
	}
	data.Overview = overview
	data.Rows = make([]repositoryEvidenceRow, 0, len(overview.Records))
	for _, item := range overview.Records {
		observedAt := "unavailable"
		if !item.Record.ObservedAt.IsZero() {
			observedAt = item.Record.ObservedAt.UTC().Format(time.RFC3339)
		}
		data.Rows = append(data.Rows, repositoryEvidenceRow{
			Record:        item.Record,
			WorkItemPath:  item.WorkItemPath,
			WorkItemTitle: item.WorkItemTitle,
			ObservedAt:    observedAt,
		})
	}
	s.renderRepositoryEvidence(w, data)
}

func (s *HostServer) renderRepositoryEvidence(w http.ResponseWriter, data repositoryEvidencePageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := s.tmpl.ExecuteTemplate(w, "repository_evidence.html", data); err != nil {
		http.Error(w, "render repository evidence", http.StatusInternalServerError)
	}
}
