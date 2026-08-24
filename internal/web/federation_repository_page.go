package web

import (
	"net/http"
	"strings"

	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/federationruntime"
)

type federationRepositoryPageData struct {
	Hostname string
	Group    federation.RepositoryGroup
	Instance federation.SourcedInstance
	Host     federationruntime.HostStatus
	Local    bool
}

func (s *HostServer) federationRepositoryPage(reader FederationReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if reader == nil {
			http.Error(w, "federation projection unavailable", http.StatusServiceUnavailable)
			return
		}

		hostID := strings.TrimSpace(r.URL.Query().Get("host"))
		instanceID := strings.TrimSpace(r.URL.Query().Get("instance"))
		if hostID == "" || instanceID == "" {
			http.Error(w, "federation host and instance are required", http.StatusBadRequest)
			return
		}

		projection, err := reader.Build(r.Context())
		if err != nil {
			http.Error(w, "load federation projection: "+err.Error(), http.StatusServiceUnavailable)
			return
		}

		var host federationruntime.HostStatus
		hostFound := false
		for _, candidate := range projection.Hosts {
			if candidate.HostID == hostID {
				host = candidate
				hostFound = true
				break
			}
		}
		if !hostFound {
			http.NotFound(w, r)
			return
		}

		var group federation.RepositoryGroup
		var instance federation.SourcedInstance
		instanceFound := false
		for _, candidateGroup := range projection.Federation.Repositories {
			for _, candidate := range candidateGroup.Instances {
				if candidate.HostID == hostID && candidate.InstanceID == instanceID {
					group = candidateGroup
					instance = candidate
					instanceFound = true
					break
				}
			}
			if instanceFound {
				break
			}
		}
		if !instanceFound {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if err := s.tmpl.ExecuteTemplate(w, "federation_repository.html", federationRepositoryPageData{
			Hostname: s.catalog.Hostname(),
			Group:    group,
			Instance: instance,
			Host:     host,
			Local:    host.Source == federationruntime.HostSourceLocal,
		}); err != nil {
			http.Error(w, "render federation repository", http.StatusInternalServerError)
		}
	}
}
