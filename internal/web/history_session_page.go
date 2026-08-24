package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sergii/specview/internal/executionhistory"
)

type historySessionPageData struct {
	Hostname    string
	Entry       executionhistory.Entry
	ProcessIDs  string
	StartedAt   string
	LastSeenAt  string
	EndedAt     string
	ObservedFor string
}

func (s *HostServer) historySessionPage(w http.ResponseWriter, r *http.Request) {
	repositoryID := strings.TrimSpace(r.URL.Query().Get("repository"))
	sessionID := strings.TrimSpace(r.URL.Query().Get("session"))
	entry, ok, err := executionhistory.NewReader(s.catalog).Find(r.Context(), repositoryID, sessionID)
	if err != nil {
		http.Error(w, "load execution session: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}

	processIDs := make([]string, 0, len(entry.ProcessIDs))
	for _, processID := range entry.ProcessIDs {
		processIDs = append(processIDs, strconv.Itoa(processID))
	}
	endedAt := "Active"
	end := entry.LastSeenAt
	if entry.EndedAt != nil {
		endedAt = formatSessionTime(*entry.EndedAt)
		end = *entry.EndedAt
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := s.tmpl.ExecuteTemplate(w, "history_session.html", historySessionPageData{
		Hostname:    s.catalog.Hostname(),
		Entry:       entry,
		ProcessIDs:  strings.Join(processIDs, ", "),
		StartedAt:   formatSessionTime(entry.StartedAt),
		LastSeenAt:  formatSessionTime(entry.LastSeenAt),
		EndedAt:     endedAt,
		ObservedFor: end.Sub(entry.StartedAt).Round(time.Second).String(),
	}); err != nil {
		http.Error(w, "render execution session", http.StatusInternalServerError)
	}
}

func formatSessionTime(value time.Time) string {
	return value.Format(time.RFC3339)
}
