package federationpeers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sergii/specview/internal/federation"
)

const ObservationVersion = 1

type RemoteObservation struct {
	Version       int                      `json:"version"`
	Peer          string                   `json:"peer"`
	RetrievedAt   *time.Time               `json:"retrieved_at,omitempty"`
	LastAttemptAt *time.Time               `json:"last_attempt_at,omitempty"`
	LastSuccessAt *time.Time               `json:"last_success_at,omitempty"`
	LastError     string                   `json:"last_error,omitempty"`
	Snapshot      *federation.HostSnapshot `json:"snapshot,omitempty"`
}

type Freshness string

const (
	FreshnessFresh          Freshness = "fresh"
	FreshnessStale          Freshness = "stale"
	FreshnessUnreachable    Freshness = "unreachable"
	FreshnessNeverRetrieved Freshness = "never_retrieved"
)

type PeerStatus struct {
	Peer          Peer
	Freshness     Freshness
	RetrievedAt   *time.Time
	LastAttemptAt *time.Time
	LastSuccessAt *time.Time
	LastError     string
	SourceAge     time.Duration
	Snapshot      *federation.HostSnapshot
}

type ObservationStore struct {
	dir string
}

func ObservationDir(catalogPath string) string {
	if strings.TrimSpace(catalogPath) == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(catalogPath), "federation", "peers")
}

func NewObservationStore(dir string) *ObservationStore {
	return &ObservationStore{dir: dir}
}

func (s *ObservationStore) Load(peerName string) (RemoteObservation, error) {
	if !peerNamePattern.MatchString(peerName) {
		return RemoteObservation{}, fmt.Errorf("invalid federation peer name %q", peerName)
	}
	if s == nil || strings.TrimSpace(s.dir) == "" {
		return emptyObservation(peerName), nil
	}
	data, err := os.ReadFile(s.path(peerName))
	if errors.Is(err, os.ErrNotExist) {
		return emptyObservation(peerName), nil
	}
	if err != nil {
		return RemoteObservation{}, err
	}
	observation, err := decodeObservation(data)
	if err != nil {
		return RemoteObservation{}, err
	}
	if observation.Peer != peerName {
		return RemoteObservation{}, fmt.Errorf("remote observation peer %q does not match requested peer %q", observation.Peer, peerName)
	}
	return observation, nil
}

func (s *ObservationStore) RecordSuccess(peer Peer, snapshot federation.HostSnapshot, now time.Time) (RemoteObservation, error) {
	if err := ValidatePeer(peer); err != nil {
		return RemoteObservation{}, err
	}
	if err := snapshot.Validate(); err != nil {
		return RemoteObservation{}, err
	}
	if snapshot.HostID != peer.ExpectedHostID {
		return RemoteObservation{}, fmt.Errorf("federation peer Host ID %q does not match expected %q", snapshot.HostID, peer.ExpectedHostID)
	}
	observation, err := s.Load(peer.Name)
	if err != nil {
		return RemoteObservation{}, err
	}
	now = now.UTC()
	copySnapshot := snapshot
	observation.Version = ObservationVersion
	observation.Peer = peer.Name
	observation.RetrievedAt = timePointer(now)
	observation.LastAttemptAt = timePointer(now)
	observation.LastSuccessAt = timePointer(now)
	observation.LastError = ""
	observation.Snapshot = &copySnapshot
	if err := s.save(observation); err != nil {
		return RemoteObservation{}, err
	}
	return cloneObservation(observation), nil
}

func (s *ObservationStore) RecordFailure(peer Peer, failure error, now time.Time) (RemoteObservation, error) {
	if err := ValidatePeer(peer); err != nil {
		return RemoteObservation{}, err
	}
	observation, err := s.Load(peer.Name)
	if err != nil {
		return RemoteObservation{}, err
	}
	now = now.UTC()
	observation.Version = ObservationVersion
	observation.Peer = peer.Name
	observation.LastAttemptAt = timePointer(now)
	observation.LastError = boundedError(failure)
	if err := s.save(observation); err != nil {
		return RemoteObservation{}, err
	}
	return cloneObservation(observation), nil
}

func (s *ObservationStore) Remove(peerName string) error {
	if !peerNamePattern.MatchString(peerName) {
		return fmt.Errorf("invalid federation peer name %q", peerName)
	}
	if s == nil || strings.TrimSpace(s.dir) == "" {
		return nil
	}
	err := os.Remove(s.path(peerName))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func ProjectStatus(peer Peer, observation RemoteObservation, now time.Time) PeerStatus {
	status := PeerStatus{
		Peer:          clonePeer(peer),
		RetrievedAt:   cloneTimePointer(observation.RetrievedAt),
		LastAttemptAt: cloneTimePointer(observation.LastAttemptAt),
		LastSuccessAt: cloneTimePointer(observation.LastSuccessAt),
		LastError:     observation.LastError,
	}
	if observation.Snapshot != nil {
		copySnapshot := *observation.Snapshot
		status.Snapshot = &copySnapshot
		status.SourceAge = now.UTC().Sub(copySnapshot.ObservedAt.UTC())
		if status.SourceAge < 0 {
			status.SourceAge = 0
		}
	}

	if latestAttemptFailed(observation) {
		status.Freshness = FreshnessUnreachable
		return status
	}
	if observation.Snapshot == nil {
		status.Freshness = FreshnessNeverRetrieved
		return status
	}
	staleAfter := time.Duration(peer.StaleAfterSeconds) * time.Second
	if status.SourceAge > staleAfter {
		status.Freshness = FreshnessStale
		return status
	}
	status.Freshness = FreshnessFresh
	return status
}

func decodeObservation(data []byte) (RemoteObservation, error) {
	var observation RemoteObservation
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&observation); err != nil {
		return RemoteObservation{}, fmt.Errorf("decode remote federation observation: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return RemoteObservation{}, errors.New("decode remote federation observation: multiple JSON values")
		}
		return RemoteObservation{}, fmt.Errorf("decode remote federation observation: %w", err)
	}
	if err := validateObservation(observation); err != nil {
		return RemoteObservation{}, err
	}
	return observation, nil
}

func validateObservation(observation RemoteObservation) error {
	if observation.Version != ObservationVersion {
		return fmt.Errorf("unsupported remote federation observation version %d", observation.Version)
	}
	if !peerNamePattern.MatchString(observation.Peer) {
		return fmt.Errorf("invalid remote federation observation peer %q", observation.Peer)
	}
	if observation.Snapshot != nil {
		if err := observation.Snapshot.Validate(); err != nil {
			return err
		}
		if observation.RetrievedAt == nil || observation.LastSuccessAt == nil {
			return errors.New("remote federation observation with snapshot requires retrieved_at and last_success_at")
		}
	}
	if observation.LastError != "" && observation.LastAttemptAt == nil {
		return errors.New("remote federation observation last_error requires last_attempt_at")
	}
	return nil
}

func (s *ObservationStore) save(observation RemoteObservation) error {
	if s == nil || strings.TrimSpace(s.dir) == "" {
		return nil
	}
	if err := validateObservation(observation); err != nil {
		return err
	}
	data, err := json.MarshalIndent(observation, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(s.path(observation.Peer), append(data, '\n'))
}

func (s *ObservationStore) path(peerName string) string {
	return filepath.Join(s.dir, peerName+".json")
}

func emptyObservation(peerName string) RemoteObservation {
	return RemoteObservation{Version: ObservationVersion, Peer: peerName}
}

func latestAttemptFailed(observation RemoteObservation) bool {
	if observation.LastError == "" || observation.LastAttemptAt == nil {
		return false
	}
	return observation.LastSuccessAt == nil || !observation.LastAttemptAt.Before(*observation.LastSuccessAt)
}

func boundedError(err error) string {
	if err == nil {
		return "federation peer retrieval failed"
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		message = "federation peer retrieval failed"
	}
	const maxBytes = 1024
	if len(message) > maxBytes {
		message = message[:maxBytes]
	}
	return message
}

func timePointer(value time.Time) *time.Time {
	copyValue := value
	return &copyValue
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func cloneObservation(observation RemoteObservation) RemoteObservation {
	copyObservation := observation
	copyObservation.RetrievedAt = cloneTimePointer(observation.RetrievedAt)
	copyObservation.LastAttemptAt = cloneTimePointer(observation.LastAttemptAt)
	copyObservation.LastSuccessAt = cloneTimePointer(observation.LastSuccessAt)
	if observation.Snapshot != nil {
		copySnapshot := *observation.Snapshot
		copyObservation.Snapshot = &copySnapshot
	}
	return copyObservation
}
