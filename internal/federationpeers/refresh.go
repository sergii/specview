package federationpeers

import (
	"context"
	"fmt"
	"time"

	"github.com/sergii/specview/internal/federationhttp"
)

type Refresher struct {
	client *federationhttp.Client
	store  *ObservationStore
	now    func() time.Time
}

func NewRefresher(client *federationhttp.Client, store *ObservationStore) *Refresher {
	if client == nil {
		client = federationhttp.NewClient()
	}
	return &Refresher{client: client, store: store, now: time.Now}
}

func (r *Refresher) Refresh(ctx context.Context, peer Peer) (PeerStatus, error) {
	if r == nil || r.store == nil {
		return PeerStatus{}, fmt.Errorf("federation peer observation store is required")
	}
	if err := ValidatePeer(peer); err != nil {
		return PeerStatus{}, err
	}

	now := r.now().UTC()
	headers, err := ResolveCredentialHeaders(peer.Credentials)
	if err != nil {
		observation, recordErr := r.store.RecordFailure(peer, err, now)
		if recordErr != nil {
			return PeerStatus{}, recordErr
		}
		return ProjectStatus(peer, observation, now), err
	}

	snapshot, err := r.client.FetchWithHeaders(ctx, peer.URL, peer.ExpectedHostID, headers)
	if err != nil {
		observation, recordErr := r.store.RecordFailure(peer, err, now)
		if recordErr != nil {
			return PeerStatus{}, recordErr
		}
		return ProjectStatus(peer, observation, now), err
	}
	observation, err := r.store.RecordSuccess(peer, snapshot, now)
	if err != nil {
		return PeerStatus{}, err
	}
	return ProjectStatus(peer, observation, now), nil
}

func (r *Refresher) Status(peer Peer) (PeerStatus, error) {
	if r == nil || r.store == nil {
		return PeerStatus{}, fmt.Errorf("federation peer observation store is required")
	}
	observation, err := r.store.Load(peer.Name)
	if err != nil {
		return PeerStatus{}, err
	}
	return ProjectStatus(peer, observation, r.now().UTC()), nil
}
