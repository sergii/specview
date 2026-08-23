package federationpeers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/sergii/specview/internal/federationhttp"
	"github.com/sergii/specview/internal/identity"
)

const (
	RegistryVersion          = 1
	DefaultStaleAfterSeconds = 300
	maxCredentialHeaders     = 16
)

var (
	peerNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,62}$`)
	envNamePattern  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

type CredentialRef struct {
	Type    string            `json:"type"`
	Headers map[string]string `json:"headers"`
}

type Peer struct {
	Name              string         `json:"name"`
	URL               string         `json:"url"`
	ExpectedHostID    string         `json:"expected_host_id"`
	StaleAfterSeconds int            `json:"stale_after_seconds"`
	Credentials       *CredentialRef `json:"credentials,omitempty"`
}

type persistedRegistry struct {
	Version int    `json:"version"`
	Peers   []Peer `json:"peers"`
}

type Registry struct {
	path string

	mu    sync.RWMutex
	peers map[string]Peer
}

func RegistryPath(catalogPath string) string {
	if strings.TrimSpace(catalogPath) == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(catalogPath), "federation-peers.json")
}

func OpenRegistry(path string) (*Registry, error) {
	r := &Registry{path: path, peers: make(map[string]Peer)}
	if strings.TrimSpace(path) == "" {
		return r, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return r, nil
	}
	if err != nil {
		return nil, err
	}
	persisted, err := decodeRegistry(data)
	if err != nil {
		return nil, err
	}
	for _, peer := range persisted.Peers {
		if err := ValidatePeer(peer); err != nil {
			return nil, fmt.Errorf("invalid federation peer %q: %w", peer.Name, err)
		}
		if _, exists := r.peers[peer.Name]; exists {
			return nil, fmt.Errorf("duplicate federation peer %q", peer.Name)
		}
		r.peers[peer.Name] = clonePeer(peer)
	}
	return r, nil
}

func decodeRegistry(data []byte) (persistedRegistry, error) {
	var persisted persistedRegistry
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&persisted); err != nil {
		return persistedRegistry{}, fmt.Errorf("decode federation peer registry: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return persistedRegistry{}, errors.New("decode federation peer registry: multiple JSON values")
		}
		return persistedRegistry{}, fmt.Errorf("decode federation peer registry: %w", err)
	}
	if persisted.Version != RegistryVersion {
		return persistedRegistry{}, fmt.Errorf("unsupported federation peer registry version %d", persisted.Version)
	}
	return persisted, nil
}

func ValidatePeer(peer Peer) error {
	if !peerNamePattern.MatchString(peer.Name) {
		return errors.New("name must be a lowercase slug using letters, numbers, dot, underscore, or hyphen")
	}
	peerURL, err := federationhttp.ValidatePeerURL(peer.URL)
	if err != nil {
		return err
	}
	if peerURL.String() != peer.URL {
		return fmt.Errorf("URL must use canonical snapshot path %s", federationhttp.SnapshotPath)
	}
	if err := identity.ValidateHostID(strings.TrimSpace(peer.ExpectedHostID)); err != nil {
		return fmt.Errorf("expected Host ID is required: %w", err)
	}
	if peer.StaleAfterSeconds <= 0 {
		return errors.New("stale_after_seconds must be positive")
	}
	if peer.Credentials != nil {
		if err := validateCredentials(*peer.Credentials); err != nil {
			return err
		}
	}
	return nil
}

func validateCredentials(credentials CredentialRef) error {
	if credentials.Type != "env_headers" {
		return fmt.Errorf("unsupported federation credential type %q", credentials.Type)
	}
	if len(credentials.Headers) == 0 {
		return errors.New("env_headers credentials require at least one header")
	}
	if len(credentials.Headers) > maxCredentialHeaders {
		return fmt.Errorf("env_headers credentials support at most %d headers", maxCredentialHeaders)
	}
	for headerName, envName := range credentials.Headers {
		if !validCredentialHeaderName(headerName) {
			return fmt.Errorf("credential header %q is not allowed", headerName)
		}
		if !envNamePattern.MatchString(envName) {
			return fmt.Errorf("credential environment variable name %q is invalid", envName)
		}
	}
	return nil
}

func validCredentialHeaderName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || !isHTTPToken(name) {
		return false
	}
	switch strings.ToLower(http.CanonicalHeaderKey(name)) {
	case "accept", "connection", "content-length", "host", "proxy-connection", "te", "trailer", "transfer-encoding", "upgrade":
		return false
	default:
		return true
	}
}

func isHTTPToken(value string) bool {
	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			continue
		}
		switch c {
		case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
			continue
		default:
			return false
		}
	}
	return true
}

func (r *Registry) Add(peer Peer) error {
	if err := ValidatePeer(peer); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.peers[peer.Name]; exists {
		return fmt.Errorf("federation peer %q already exists", peer.Name)
	}
	r.peers[peer.Name] = clonePeer(peer)
	if err := r.saveLocked(); err != nil {
		delete(r.peers, peer.Name)
		return err
	}
	return nil
}

func (r *Registry) Remove(name string) error {
	name = strings.TrimSpace(name)
	r.mu.Lock()
	defer r.mu.Unlock()
	peer, exists := r.peers[name]
	if !exists {
		return fmt.Errorf("federation peer %q not found", name)
	}
	delete(r.peers, name)
	if err := r.saveLocked(); err != nil {
		r.peers[name] = peer
		return err
	}
	return nil
}

func (r *Registry) Find(name string) (Peer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	peer, ok := r.peers[strings.TrimSpace(name)]
	if !ok {
		return Peer{}, false
	}
	return clonePeer(peer), true
}

func (r *Registry) List() []Peer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	peers := make([]Peer, 0, len(r.peers))
	for _, peer := range r.peers {
		peers = append(peers, clonePeer(peer))
	}
	sort.Slice(peers, func(i, j int) bool { return peers[i].Name < peers[j].Name })
	return peers
}

func (r *Registry) saveLocked() error {
	if strings.TrimSpace(r.path) == "" {
		return nil
	}
	peers := make([]Peer, 0, len(r.peers))
	for _, peer := range r.peers {
		peers = append(peers, clonePeer(peer))
	}
	sort.Slice(peers, func(i, j int) bool { return peers[i].Name < peers[j].Name })
	data, err := json.MarshalIndent(persistedRegistry{Version: RegistryVersion, Peers: peers}, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(r.path, append(data, '\n'))
}

func clonePeer(peer Peer) Peer {
	copyPeer := peer
	if peer.Credentials != nil {
		credentials := *peer.Credentials
		credentials.Headers = make(map[string]string, len(peer.Credentials.Headers))
		for name, envName := range peer.Credentials.Headers {
			credentials.Headers[name] = envName
		}
		copyPeer.Credentials = &credentials
	}
	return copyPeer
}

func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
