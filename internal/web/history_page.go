package web

import (
	"net/http"

	"github.com/sergii/specview/internal/executionhistory"
)

type historyPageData struct {
	Hostname   string
	Projection executionhistory.Projection
}

func (s *HostServer) historyPage(w http.ResponseWriter, r *http.Request) {
	projection, err := executionhistory.NewReader(s.catalog).Build(r.Context())
	if err != nil {
		http.Error(w, "load execution history: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := s.tmpl.ExecuteTemplate(w, "history.html", historyPageData{
		Hostname:   s.catalog.Hostname(),
		Projection: projection,
	}); err != nil {
		http.Error(w, "render execution history", http.StatusInternalServerError)
	}
}
