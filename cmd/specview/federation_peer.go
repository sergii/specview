package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/federationhttp"
	"github.com/sergii/specview/internal/federationpeers"
	"github.com/sergii/specview/internal/hoststate"
)

const peerUsage = "usage: specview federation peer <add|list|show|refresh|remove> ..."

func runFederationPeer(args []string) error {
	if len(args) == 0 {
		return errors.New(peerUsage)
	}
	registry, store, err := openFederationPeerState()
	if err != nil {
		return err
	}

	switch args[0] {
	case "add":
		return addFederationPeer(registry, args[1:])
	case "list":
		if len(args) != 1 {
			return errors.New("usage: specview federation peer list")
		}
		return listFederationPeers(registry, store)
	case "show":
		if len(args) != 2 {
			return errors.New("usage: specview federation peer show <name>")
		}
		return showFederationPeer(registry, store, args[1])
	case "refresh":
		if len(args) != 2 {
			return errors.New("usage: specview federation peer refresh <name>")
		}
		return refreshFederationPeer(registry, store, args[1])
	case "remove":
		if len(args) != 2 {
			return errors.New("usage: specview federation peer remove <name>")
		}
		return removeFederationPeer(registry, store, args[1])
	default:
		return fmt.Errorf("unknown federation peer command %q", args[0])
	}
}

func openFederationPeerState() (*federationpeers.Registry, *federationpeers.ObservationStore, error) {
	catalogPath, err := hoststate.DefaultStatePath()
	if err != nil {
		return nil, nil, err
	}
	registry, err := federationpeers.OpenRegistry(federationpeers.RegistryPath(catalogPath))
	if err != nil {
		return nil, nil, err
	}
	store := federationpeers.NewObservationStore(federationpeers.ObservationDir(catalogPath))
	return registry, store, nil
}

func addFederationPeer(registry *federationpeers.Registry, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: specview federation peer add <name> --url <url> --host <host:id> [--stale-after 5m] [--header-env Header=ENV]...")
	}
	peer := federationpeers.Peer{
		Name:              strings.TrimSpace(args[0]),
		StaleAfterSeconds: federationpeers.DefaultStaleAfterSeconds,
	}
	headerRefs := map[string]string{}
	for i := 1; i < len(args); i++ {
		arg := args[i]
		var value string
		var ok bool
		switch {
		case arg == "--url":
			value, ok = nextPeerFlagValue(args, &i)
			if !ok || peer.URL != "" {
				return errors.New("peer add requires exactly one --url")
			}
			peer.URL = value
		case strings.HasPrefix(arg, "--url="):
			if peer.URL != "" {
				return errors.New("peer add requires exactly one --url")
			}
			peer.URL = strings.TrimSpace(strings.TrimPrefix(arg, "--url="))
		case arg == "--host":
			value, ok = nextPeerFlagValue(args, &i)
			if !ok || peer.ExpectedHostID != "" {
				return errors.New("peer add requires exactly one --host")
			}
			peer.ExpectedHostID = value
		case strings.HasPrefix(arg, "--host="):
			if peer.ExpectedHostID != "" {
				return errors.New("peer add requires exactly one --host")
			}
			peer.ExpectedHostID = strings.TrimSpace(strings.TrimPrefix(arg, "--host="))
		case arg == "--stale-after":
			value, ok = nextPeerFlagValue(args, &i)
			if !ok {
				return errors.New("--stale-after requires a duration")
			}
			seconds, err := parseStaleAfter(value)
			if err != nil {
				return err
			}
			peer.StaleAfterSeconds = seconds
		case strings.HasPrefix(arg, "--stale-after="):
			seconds, err := parseStaleAfter(strings.TrimPrefix(arg, "--stale-after="))
			if err != nil {
				return err
			}
			peer.StaleAfterSeconds = seconds
		case arg == "--header-env":
			value, ok = nextPeerFlagValue(args, &i)
			if !ok {
				return errors.New("--header-env requires Header=ENV")
			}
			if err := addHeaderEnvRef(headerRefs, value); err != nil {
				return err
			}
		case strings.HasPrefix(arg, "--header-env="):
			if err := addHeaderEnvRef(headerRefs, strings.TrimPrefix(arg, "--header-env=")); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown peer add argument %q", arg)
		}
	}
	if peer.URL == "" || peer.ExpectedHostID == "" {
		return errors.New("peer add requires --url and --host")
	}
	peerURL, err := federationhttp.ValidatePeerURL(peer.URL)
	if err != nil {
		return err
	}
	peer.URL = peerURL.String()
	if len(headerRefs) > 0 {
		peer.Credentials = &federationpeers.CredentialRef{Type: "env_headers", Headers: headerRefs}
	}
	if err := registry.Add(peer); err != nil {
		return err
	}
	fmt.Printf("Added federation peer %s\n", peer.Name)
	return nil
}

func listFederationPeers(registry *federationpeers.Registry, store *federationpeers.ObservationStore) error {
	refresher := federationpeers.NewRefresher(nil, store)
	peers := registry.List()
	outputs := make([]peerStatusOutput, 0, len(peers))
	for _, peer := range peers {
		status, err := refresher.Status(peer)
		if err != nil {
			return err
		}
		outputs = append(outputs, makePeerStatusOutput(status))
	}
	return writeFederationJSON(os.Stdout, outputs)
}

func showFederationPeer(registry *federationpeers.Registry, store *federationpeers.ObservationStore, name string) error {
	peer, ok := registry.Find(strings.TrimSpace(name))
	if !ok {
		return fmt.Errorf("federation peer %q not found", name)
	}
	status, err := federationpeers.NewRefresher(nil, store).Status(peer)
	if err != nil {
		return err
	}
	return writeFederationJSON(os.Stdout, makePeerStatusOutput(status))
}

func refreshFederationPeer(registry *federationpeers.Registry, store *federationpeers.ObservationStore, name string) error {
	peer, ok := registry.Find(strings.TrimSpace(name))
	if !ok {
		return fmt.Errorf("federation peer %q not found", name)
	}
	status, refreshErr := federationpeers.NewRefresher(nil, store).Refresh(context.Background(), peer)
	if err := writeFederationJSON(os.Stdout, makePeerStatusOutput(status)); err != nil {
		return err
	}
	if refreshErr != nil {
		return fmt.Errorf("refresh federation peer %s: %w", peer.Name, refreshErr)
	}
	return nil
}

func removeFederationPeer(registry *federationpeers.Registry, store *federationpeers.ObservationStore, name string) error {
	name = strings.TrimSpace(name)
	if err := registry.Remove(name); err != nil {
		return err
	}
	if err := store.Remove(name); err != nil {
		return err
	}
	fmt.Printf("Removed federation peer %s\n", name)
	return nil
}

func nextPeerFlagValue(args []string, index *int) (string, bool) {
	if *index+1 >= len(args) {
		return "", false
	}
	*index = *index + 1
	value := strings.TrimSpace(args[*index])
	return value, value != ""
}

func parseStaleAfter(raw string) (int, error) {
	duration, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil || duration <= 0 || duration%time.Second != 0 {
		return 0, fmt.Errorf("invalid --stale-after %q; use a positive whole-second duration such as 5m or 1h", raw)
	}
	seconds := duration / time.Second
	if seconds > time.Duration(int(^uint(0)>>1)) {
		return 0, errors.New("--stale-after is too large")
	}
	return int(seconds), nil
}

func addHeaderEnvRef(refs map[string]string, raw string) error {
	name, envName, ok := strings.Cut(strings.TrimSpace(raw), "=")
	name = strings.TrimSpace(name)
	envName = strings.TrimSpace(envName)
	if !ok || name == "" || envName == "" {
		return fmt.Errorf("invalid --header-env %q; want Header=ENV", raw)
	}
	if _, exists := refs[name]; exists {
		return fmt.Errorf("duplicate --header-env header %q", name)
	}
	refs[name] = envName
	return nil
}

type peerStatusOutput struct {
	Name              string                    `json:"name"`
	URL               string                    `json:"url"`
	ExpectedHostID    string                    `json:"expected_host_id"`
	StaleAfterSeconds int                       `json:"stale_after_seconds"`
	CredentialType    string                    `json:"credential_type,omitempty"`
	CredentialRefs    map[string]string         `json:"credential_refs,omitempty"`
	Status            federationpeers.Freshness `json:"status"`
	RetrievedAt       *time.Time                `json:"retrieved_at,omitempty"`
	LastAttemptAt     *time.Time                `json:"last_attempt_at,omitempty"`
	LastSuccessAt     *time.Time                `json:"last_success_at,omitempty"`
	LastError         string                    `json:"last_error,omitempty"`
	SourceAgeSeconds  int64                     `json:"source_age_seconds,omitempty"`
	Snapshot          *federation.HostSnapshot  `json:"snapshot,omitempty"`
}

func makePeerStatusOutput(status federationpeers.PeerStatus) peerStatusOutput {
	output := peerStatusOutput{
		Name:              status.Peer.Name,
		URL:               status.Peer.URL,
		ExpectedHostID:    status.Peer.ExpectedHostID,
		StaleAfterSeconds: status.Peer.StaleAfterSeconds,
		Status:            status.Freshness,
		RetrievedAt:       status.RetrievedAt,
		LastAttemptAt:     status.LastAttemptAt,
		LastSuccessAt:     status.LastSuccessAt,
		LastError:         status.LastError,
		SourceAgeSeconds:  int64(status.SourceAge / time.Second),
		Snapshot:          status.Snapshot,
	}
	if status.Peer.Credentials != nil {
		output.CredentialType = status.Peer.Credentials.Type
		output.CredentialRefs = make(map[string]string, len(status.Peer.Credentials.Headers))
		keys := make([]string, 0, len(status.Peer.Credentials.Headers))
		for name := range status.Peer.Credentials.Headers {
			keys = append(keys, name)
		}
		sort.Strings(keys)
		for _, name := range keys {
			output.CredentialRefs[name] = status.Peer.Credentials.Headers[name]
		}
	}
	return output
}
