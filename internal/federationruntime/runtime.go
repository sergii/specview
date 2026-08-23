package federationruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/federationpeers"
)

const DefaultPollInterval = 30 * time.Second

type PeerRefresher interface {
	Refresh(context.Context, federationpeers.Peer) (federationpeers.PeerStatus, error)
}

type RefreshResult struct {
	Peer      string                    `json:"peer"`
	Freshness federationpeers.Freshness `json:"freshness"`
	Error     string                    `json:"error,omitempty"`
}

type RefreshSummary struct {
	Peers     int             `json:"peers"`
	Succeeded int             `json:"succeeded"`
	Failed    int             `json:"failed"`
	Results   []RefreshResult `json:"results"`
}

type Poller struct {
	registryPath string
	refresher    PeerRefresher
	interval     time.Duration
	onChange     func()

	mu              sync.Mutex
	lastMaterialKey string
	seenMaterial    bool
}

func NewPoller(registryPath string, refresher PeerRefresher, interval time.Duration, onChange func()) (*Poller, error) {
	if refresher == nil {
		return nil, errors.New("federation peer refresher is required")
	}
	if interval <= 0 {
		return nil, errors.New("federation peer poll interval must be positive")
	}
	return &Poller{
		registryPath: strings.TrimSpace(registryPath),
		refresher:    refresher,
		interval:     interval,
		onChange:     onChange,
	}, nil
}

func (p *Poller) Refresh(ctx context.Context) (RefreshSummary, error) {
	if p == nil || p.refresher == nil {
		return RefreshSummary{}, errors.New("federation peer poller is not configured")
	}
	registry, err := federationpeers.OpenRegistry(p.registryPath)
	if err != nil {
		return RefreshSummary{}, err
	}
	peers := registry.List()
	summary := RefreshSummary{
		Peers:   len(peers),
		Results: make([]RefreshResult, 0, len(peers)),
	}
	materials := make([]peerMaterial, 0, len(peers))

	for _, peer := range peers {
		if err := ctx.Err(); err != nil {
			return summary, err
		}
		status, refreshErr := p.refresher.Refresh(ctx, peer)
		result := RefreshResult{Peer: peer.Name, Freshness: status.Freshness}
		if refreshErr != nil {
			if ctxErr := ctx.Err(); ctxErr != nil && (errors.Is(refreshErr, context.Canceled) || errors.Is(refreshErr, context.DeadlineExceeded)) {
				return summary, ctxErr
			}
			result.Error = refreshErr.Error()
			summary.Failed++
		} else {
			summary.Succeeded++
		}
		summary.Results = append(summary.Results, result)
		materials = append(materials, peerMaterial{
			Peer:      peer,
			Freshness: status.Freshness,
			LastError: status.LastError,
			Error:     result.Error,
			Snapshot:  status.Snapshot,
		})
	}

	if p.onChange != nil && p.updateMaterialKey(materialKey(materials)) {
		p.onChange()
	}
	return summary, nil
}

func (p *Poller) Run(ctx context.Context) {
	if p == nil {
		return
	}
	slog.Info("federation peer runtime started", "interval", p.interval)
	defer slog.Info("federation peer runtime stopped")

	if summary, err := p.Refresh(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Warn("initial federation peer refresh failed", "error", err)
	} else {
		logRefreshSummary("initial", summary)
	}

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			summary, err := p.Refresh(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				slog.Warn("federation peer refresh cycle failed", "error", err)
				continue
			}
			logRefreshSummary("periodic", summary)
		}
	}
}

type peerMaterial struct {
	Peer      federationpeers.Peer      `json:"peer"`
	Freshness federationpeers.Freshness `json:"freshness"`
	LastError string                    `json:"last_error,omitempty"`
	Error     string                    `json:"error,omitempty"`
	Snapshot  *federation.HostSnapshot  `json:"snapshot,omitempty"`
}

func materialKey(materials []peerMaterial) string {
	data, err := json.Marshal(materials)
	if err != nil {
		return "material-encode-error"
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (p *Poller) updateMaterialKey(key string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	changed := !p.seenMaterial || p.lastMaterialKey != key
	p.seenMaterial = true
	p.lastMaterialKey = key
	return changed
}

func logRefreshSummary(kind string, summary RefreshSummary) {
	if summary.Failed > 0 {
		slog.Warn("federation peer refresh completed with failures",
			"kind", kind,
			"peers", summary.Peers,
			"succeeded", summary.Succeeded,
			"failed", summary.Failed,
		)
		return
	}
	slog.Debug("federation peer refresh completed",
		"kind", kind,
		"peers", summary.Peers,
		"succeeded", summary.Succeeded,
	)
}
