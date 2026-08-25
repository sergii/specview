package web

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/sergii/specview/internal/controlplane"
	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/federationruntime"
)

type federationHostRepositoryData struct {
	Group    federation.RepositoryGroup
	Instance federation.SourcedInstance
	Href     string
}

type federationHostAttentionData struct {
	Attention controlplane.HostAttentionSummary
	Href      string
}

type federationHostPageData struct {
	Hostname     string
	Host         federationruntime.HostStatus
	Repositories []federationHostRepositoryData
	Attention    []federationHostAttentionData
}

func (s *HostServer) federationHostPage(reader FederationReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if reader == nil {
			http.Error(w, "federation projection unavailable", http.StatusServiceUnavailable)
			return
		}

		hostID := strings.TrimSpace(r.URL.Query().Get("host"))
		if hostID == "" {
			http.Error(w, "federation host is required", http.StatusBadRequest)
			return
		}

		projection, err := reader.Build(r.Context())
		if err != nil {
			http.Error(w, "load federation projection: "+err.Error(), http.StatusServiceUnavailable)
			return
		}

		host, found := federationruntime.SelectHost(projection, hostID)
		if !found {
			http.NotFound(w, r)
			return
		}

		repositories := federationHostRepositories(projection, hostID)
		attention := federationHostAttention(host, repositories)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if err := s.tmpl.ExecuteTemplate(w, "federation_host.html", federationHostPageData{
			Hostname:     s.catalog.Hostname(),
			Host:         host,
			Repositories: repositories,
			Attention:    attention,
		}); err != nil {
			http.Error(w, "render federation Host", http.StatusInternalServerError)
		}
	}
}

func federationHostRepositories(projection federationruntime.Projection, hostID string) []federationHostRepositoryData {
	selections := federationruntime.RepositoriesForHost(projection, hostID)
	result := make([]federationHostRepositoryData, 0, len(selections))
	for _, selection := range selections {
		result = append(result, federationHostRepositoryData{
			Group:    selection.Group,
			Instance: selection.Instance,
			Href:     federationRepositoryHref(hostID, selection.Instance.InstanceID),
		})
	}
	return result
}

func federationHostAttention(host federationruntime.HostStatus, repositories []federationHostRepositoryData) []federationHostAttentionData {
	if host.ControlPlane == nil {
		return nil
	}
	result := make([]federationHostAttentionData, 0, len(host.ControlPlane.Attention))
	for _, item := range host.ControlPlane.Attention {
		row := federationHostAttentionData{Attention: item}
		matches := 0
		for _, repository := range repositories {
			if repository.Instance.SourceRepositoryID != item.RepositoryID {
				continue
			}
			matches++
			row.Href = repository.Href
		}
		if matches != 1 {
			row.Href = ""
		}
		result = append(result, row)
	}
	return result
}

func federationRepositoryHref(hostID, instanceID string) string {
	query := url.Values{"host": {hostID}, "instance": {instanceID}}
	return "/federation/repository?" + query.Encode()
}
