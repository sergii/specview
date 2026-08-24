package web

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/projectstate"
	"github.com/sergii/specview/internal/sourcecontrol"
)

type repositoryAcceptancePageData struct {
	Hostname      string
	Repository    hoststate.Repository
	Overview      projectstate.AcceptanceOverview
	RevisionLabel string
	Error         string
}

func (s *HostServer) repositoryAcceptancePage(w http.ResponseWriter, r *http.Request) {
	repositoryID := strings.TrimSpace(r.URL.Query().Get("id"))
	repository, ok := s.catalog.Find(repositoryID)
	if !ok {
		http.NotFound(w, r)
		return
	}

	data := repositoryAcceptancePageData{
		Hostname:   s.catalog.Hostname(),
		Repository: repository,
	}
	project, err := projectstate.Resolve(repository.Root)
	if err != nil {
		data.Error = err.Error()
		s.renderRepositoryAcceptance(w, data)
		return
	}

	var git sourcecontrol.GitContext
	if len(project.Policy.Required) > 0 {
		repositoryContext, inspectErr := s.sourceControl.Inspect(r.Context(), repository.Root)
		if inspectErr != nil {
			data.Error = fmt.Sprintf("source-control context unavailable: %v", inspectErr)
			s.renderRepositoryAcceptance(w, data)
			return
		}
		git = repositoryContext.Git
	}

	overview, err := project.AcceptanceOverview(git)
	if err != nil {
		data.Error = err.Error()
		s.renderRepositoryAcceptance(w, data)
		return
	}
	data.Overview = overview
	if overview.Configured {
		data.RevisionLabel = (workItemAcceptanceData{Revision: overview.Revision}).RevisionLabel()
	}
	s.renderRepositoryAcceptance(w, data)
}

func (s *HostServer) renderRepositoryAcceptance(w http.ResponseWriter, data repositoryAcceptancePageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := s.tmpl.ExecuteTemplate(w, "repository_acceptance.html", data); err != nil {
		http.Error(w, "render repository acceptance", http.StatusInternalServerError)
	}
}
