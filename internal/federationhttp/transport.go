package federationhttp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/identity"
)

const (
	SnapshotPath    = "/v1/federation/snapshot"
	SnapshotPathV1  = SnapshotPath
	SnapshotPathV2  = "/v2/federation/snapshot"
	DefaultAddress  = "127.0.0.1:7332"
	DefaultMaxBytes = int64(16 << 20)
	DefaultTimeout  = 10 * time.Second
)

var errSnapshotVersionUnavailable = errors.New("federation snapshot version unavailable")

type SnapshotSource interface {
	Build(context.Context) (federation.HostSnapshot, error)
}

type Handler struct {
	source SnapshotSource
}

func NewHandler(source SnapshotSource) (*Handler, error) {
	if source == nil {
		return nil, errors.New("federation HTTP snapshot source is required")
	}
	return &Handler{source: source}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != SnapshotPathV1 && r.URL.Path != SnapshotPathV2 {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	snapshot, err := h.source.Build(r.Context())
	if err != nil {
		http.Error(w, "federation snapshot unavailable", http.StatusServiceUnavailable)
		return
	}
	if r.URL.Path == SnapshotPathV1 {
		snapshot = snapshot.V1()
	}
	if err := snapshot.Validate(); err != nil {
		http.Error(w, "federation snapshot invalid", http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		http.Error(w, "encode federation snapshot", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(append(data, '\n'))
}

type Client struct {
	httpClient *http.Client
	maxBytes   int64
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return errors.New("federation HTTP redirects are not allowed")
			},
		},
		maxBytes: DefaultMaxBytes,
	}
}

func NewClientWithHTTPClient(httpClient *http.Client, maxBytes int64) (*Client, error) {
	if httpClient == nil {
		return nil, errors.New("federation HTTP client is required")
	}
	if maxBytes <= 0 {
		return nil, errors.New("federation HTTP max response bytes must be positive")
	}
	copyClient := *httpClient
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return errors.New("federation HTTP redirects are not allowed")
	}
	return &Client{httpClient: &copyClient, maxBytes: maxBytes}, nil
}

func (c *Client) Fetch(ctx context.Context, rawURL, expectedHostID string) (federation.HostSnapshot, error) {
	return c.FetchWithHeaders(ctx, rawURL, expectedHostID, nil)
}

func (c *Client) FetchWithHeaders(ctx context.Context, rawURL, expectedHostID string, headers http.Header) (federation.HostSnapshot, error) {
	if c == nil || c.httpClient == nil {
		return federation.HostSnapshot{}, errors.New("federation HTTP client is required")
	}
	peerURL, err := ValidatePeerURL(rawURL)
	if err != nil {
		return federation.HostSnapshot{}, err
	}
	expectedHostID = strings.TrimSpace(expectedHostID)
	if expectedHostID != "" {
		if err := identity.ValidateHostID(expectedHostID); err != nil {
			return federation.HostSnapshot{}, fmt.Errorf("invalid expected federation Host ID: %w", err)
		}
	}

	var lastUnavailable error
	for _, candidate := range preferredSnapshotURLs(peerURL) {
		snapshot, fetchErr := c.fetchSnapshot(ctx, candidate, headers)
		if fetchErr != nil {
			if errors.Is(fetchErr, errSnapshotVersionUnavailable) {
				lastUnavailable = fetchErr
				continue
			}
			return federation.HostSnapshot{}, fetchErr
		}
		if expectedHostID != "" && snapshot.HostID != expectedHostID {
			return federation.HostSnapshot{}, fmt.Errorf("federation peer Host ID %q does not match expected %q", snapshot.HostID, expectedHostID)
		}
		return snapshot, nil
	}
	if lastUnavailable != nil {
		return federation.HostSnapshot{}, lastUnavailable
	}
	return federation.HostSnapshot{}, errors.New("federation snapshot unavailable")
}

func (c *Client) fetchSnapshot(ctx context.Context, peerURL *url.URL, headers http.Header) (federation.HostSnapshot, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, peerURL.String(), nil)
	if err != nil {
		return federation.HostSnapshot{}, err
	}
	request.Header.Set("Accept", "application/json")
	for name, values := range headers {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return federation.HostSnapshot{}, fmt.Errorf("fetch federation snapshot: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return federation.HostSnapshot{}, fmt.Errorf("%w: %s", errSnapshotVersionUnavailable, peerURL.Path)
	}
	if response.StatusCode != http.StatusOK {
		return federation.HostSnapshot{}, fmt.Errorf("fetch federation snapshot: unexpected HTTP status %d", response.StatusCode)
	}
	if err := requireJSONContentType(response.Header.Get("Content-Type")); err != nil {
		return federation.HostSnapshot{}, err
	}

	body, err := readBounded(response.Body, c.maxBytes)
	if err != nil {
		return federation.HostSnapshot{}, err
	}
	snapshot, err := federation.DecodeSnapshot(body)
	if err != nil {
		return federation.HostSnapshot{}, err
	}
	return snapshot, nil
}

func preferredSnapshotURLs(peerURL *url.URL) []*url.URL {
	v2 := *peerURL
	v2.Path = SnapshotPathV2
	v1 := *peerURL
	v1.Path = SnapshotPathV1
	return []*url.URL{&v2, &v1}
}

func ValidatePeerURL(rawURL string) (*url.URL, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, errors.New("federation peer URL is required")
	}
	peerURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse federation peer URL: %w", err)
	}
	if peerURL.Scheme == "" || peerURL.Host == "" {
		return nil, errors.New("federation peer URL must be absolute")
	}
	if peerURL.User != nil {
		return nil, errors.New("federation peer URL must not contain user information")
	}
	if peerURL.RawQuery != "" || peerURL.Fragment != "" {
		return nil, errors.New("federation peer URL must not contain query or fragment")
	}
	if peerURL.Path == "" || peerURL.Path == "/" {
		peerURL.Path = SnapshotPathV1
	}
	if peerURL.Path != SnapshotPathV1 && peerURL.Path != SnapshotPathV2 {
		return nil, fmt.Errorf("federation peer URL path must be %s or %s", SnapshotPathV1, SnapshotPathV2)
	}

	switch strings.ToLower(peerURL.Scheme) {
	case "https":
		return peerURL, nil
	case "http":
		if isLoopbackHost(peerURL.Hostname()) {
			return peerURL, nil
		}
		return nil, errors.New("remote federation peer URLs must use HTTPS")
	default:
		return nil, errors.New("federation peer URL must use HTTPS or loopback HTTP")
	}
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(strings.TrimSuffix(host, "."), "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func requireJSONContentType(value string) error {
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return fmt.Errorf("federation snapshot has invalid Content-Type %q", value)
	}
	if mediaType != "application/json" {
		return fmt.Errorf("federation snapshot Content-Type is %q, want application/json", mediaType)
	}
	return nil
}

func readBounded(reader io.Reader, maxBytes int64) ([]byte, error) {
	limited := io.LimitReader(reader, maxBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read federation snapshot: %w", err)
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("federation snapshot exceeds %d bytes", maxBytes)
	}
	return body, nil
}
