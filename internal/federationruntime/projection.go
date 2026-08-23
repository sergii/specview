package federationruntime

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/federationpeers"
)

const ProjectionSchemaVersion = 1

const (
	HostSourceLocal = "local"
	HostSourcePeer  = "peer"
)

type SnapshotBuilder interface {
	Build(context.Context) (federation.HostSnapshot, error)
}

type SnapshotAggregator interface {
	Aggregate(...federation.HostSnapshot) (federation.Projection, error)
}

type Projection struct {
	SchemaVersion int                   `json:"schema_version"`
	GeneratedAt   time.Time             `json:"generated_at"`
	Hosts         []HostStatus          `json:"hosts"`
	Federation    federation.Projection `json:"federation"`
}

type HostStatus struct {
	Source           string                    `json:"source"`
	Peer             string                    `json:"peer,omitempty"`
	HostID           string                    `json:"host_id"`
	Hostname         string                    `json:"hostname,omitempty"`
	Freshness        federationpeers.Freshness `json:"freshness,omitempty"`
	HasSnapshot      bool                      `json:"has_snapshot"`
	ObservedAt       *time.Time                `json:"observed_at,omitempty"`
	RetrievedAt      *time.Time                `json:"retrieved_at,omitempty"`
	LastAttemptAt    *time.Time                `json:"last_attempt_at,omitempty"`
	LastSuccessAt    *time.Time                `json:"last_success_at,omitempty"`
	LastError        string                    `json:"last_error,omitempty"`
	SourceAgeSeconds *int64                    `json:"source_age_seconds,omitempty"`
}

type ProjectionBuilder struct {
	local        SnapshotBuilder
	registryPath string
	store        *federationpeers.ObservationStore
	aggregator   SnapshotAggregator
	now          func() time.Time
}

func NewProjectionBuilder(local SnapshotBuilder, registryPath string, store *federationpeers.ObservationStore) (*ProjectionBuilder, error) {
	if local == nil {
		return nil, errors.New("local federation snapshot builder is required")
	}
	if store == nil {
		return nil, errors.New("federation peer observation store is required")
	}
	return &ProjectionBuilder{
		local:        local,
		registryPath: strings.TrimSpace(registryPath),
		store:        store,
		aggregator:   federation.NewAggregator(),
		now:          time.Now,
	}, nil
}

func (b *ProjectionBuilder) Build(ctx context.Context) (Projection, error) {
	if b == nil || b.local == nil || b.store == nil || b.aggregator == nil {
		return Projection{}, errors.New("federation multi-host projection builder is not configured")
	}

	localSnapshot, err := b.local.Build(ctx)
	if err != nil {
		return Projection{}, fmt.Errorf("build local federation snapshot: %w", err)
	}
	if err := localSnapshot.Validate(); err != nil {
		return Projection{}, fmt.Errorf("validate local federation snapshot: %w", err)
	}

	registry, err := federationpeers.OpenRegistry(b.registryPath)
	if err != nil {
		return Projection{}, fmt.Errorf("open federation peer registry: %w", err)
	}

	now := b.now().UTC()
	snapshots := []federation.HostSnapshot{localSnapshot}
	hosts := []HostStatus{localHostStatus(localSnapshot)}

	for _, peer := range registry.List() {
		observation, err := b.store.Load(peer.Name)
		if err != nil {
			return Projection{}, fmt.Errorf("load federation peer %s observation: %w", peer.Name, err)
		}
		status := federationpeers.ProjectStatus(peer, observation, now)
		hosts = append(hosts, peerHostStatus(status))
		if status.Snapshot != nil {
			snapshots = append(snapshots, *status.Snapshot)
		}
	}

	aggregated, err := b.aggregator.Aggregate(snapshots...)
	if err != nil {
		return Projection{}, fmt.Errorf("aggregate multi-host federation projection: %w", err)
	}
	// H23 owns the outer read-model clock. Align the nested derived projection to
	// the same instant without changing any H20 snapshot/correlation semantics.
	aggregated.GeneratedAt = now

	sort.SliceStable(hosts, func(i, j int) bool {
		if hosts[i].Source != hosts[j].Source {
			return hosts[i].Source == HostSourceLocal
		}
		if hosts[i].Peer != hosts[j].Peer {
			return hosts[i].Peer < hosts[j].Peer
		}
		return hosts[i].HostID < hosts[j].HostID
	})

	return Projection{
		SchemaVersion: ProjectionSchemaVersion,
		GeneratedAt:   now,
		Hosts:         hosts,
		Federation:    aggregated,
	}, nil
}

func localHostStatus(snapshot federation.HostSnapshot) HostStatus {
	observedAt := snapshot.ObservedAt.UTC()
	return HostStatus{
		Source:      HostSourceLocal,
		HostID:      snapshot.HostID,
		Hostname:    snapshot.Hostname,
		HasSnapshot: true,
		ObservedAt:  &observedAt,
	}
}

func peerHostStatus(status federationpeers.PeerStatus) HostStatus {
	host := HostStatus{
		Source:        HostSourcePeer,
		Peer:          status.Peer.Name,
		HostID:        status.Peer.ExpectedHostID,
		Freshness:     status.Freshness,
		RetrievedAt:   cloneTime(status.RetrievedAt),
		LastAttemptAt: cloneTime(status.LastAttemptAt),
		LastSuccessAt: cloneTime(status.LastSuccessAt),
		LastError:     status.LastError,
	}
	if status.Snapshot != nil {
		host.HasSnapshot = true
		host.Hostname = status.Snapshot.Hostname
		observedAt := status.Snapshot.ObservedAt.UTC()
		host.ObservedAt = &observedAt
		age := int64(status.SourceAge / time.Second)
		host.SourceAgeSeconds = &age
	}
	return host
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := value.UTC()
	return &copyValue
}
