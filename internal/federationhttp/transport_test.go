package federationhttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sergii/specview/internal/controlplane"
	"github.com/sergii/specview/internal/federation"
)

type snapshotSourceStub struct {
	snapshot federation.HostSnapshot
	err      error
	calls    int
}

func (s *snapshotSourceStub) Build(context.Context) (federation.HostSnapshot, error) {
	s.calls++
	return s.snapshot, s.err
}

func TestHandlerPreservesFrozenV1SnapshotEndpoint(t *testing.T) {
	source := &snapshotSourceStub{snapshot: validSnapshot()}
	handler, err := NewHandler(source)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, SnapshotPathV1, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%q", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	snapshot, err := federation.DecodeSnapshot(response.Body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.SchemaVersion != federation.SnapshotSchemaVersion || snapshot.ControlPlane != nil {
		t.Fatalf("v1 endpoint changed wire shape: %#v", snapshot)
	}
	if snapshot.HostID != source.snapshot.HostID || source.calls != 1 {
		t.Fatalf("unexpected served snapshot=%#v calls=%d", snapshot, source.calls)
	}
}

func TestHandlerServesV2HostControlPlane(t *testing.T) {
	source := &snapshotSourceStub{snapshot: validSnapshot()}
	handler, err := NewHandler(source)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, SnapshotPathV2, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%q", response.Code, response.Body.String())
	}
	snapshot, err := federation.DecodeSnapshot(response.Body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.SchemaVersion != federation.SnapshotSchemaVersionV2 || snapshot.ControlPlane == nil || snapshot.ControlPlane.Host != "devbox-01" {
		t.Fatalf("unexpected v2 snapshot: %#v", snapshot)
	}
}

func TestHandlerRejectsUnsupportedMethodWithoutBuildingSnapshot(t *testing.T) {
	source := &snapshotSourceStub{snapshot: validSnapshot()}
	handler, err := NewHandler(source)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, SnapshotPathV1, nil))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", response.Code)
	}
	if response.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("Allow = %q", response.Header().Get("Allow"))
	}
	if source.calls != 0 {
		t.Fatalf("snapshot source called %d times", source.calls)
	}
}

func TestHandlerDoesNotReturnPartialSnapshotOnSourceFailure(t *testing.T) {
	source := &snapshotSourceStub{err: errors.New("scan failed")}
	handler, err := NewHandler(source)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, SnapshotPathV2, nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", response.Code)
	}
	if strings.Contains(response.Body.String(), `"schema_version"`) {
		t.Fatalf("partial snapshot leaked on error: %q", response.Body.String())
	}
}

func TestClientPrefersV2AndPinsLoopbackPeer(t *testing.T) {
	source := &snapshotSourceStub{snapshot: validSnapshot()}
	handler, err := NewHandler(source)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	snapshot, err := NewClient().Fetch(context.Background(), server.URL, source.snapshot.HostID)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.HostID != source.snapshot.HostID || snapshot.SchemaVersion != federation.SnapshotSchemaVersionV2 || snapshot.ControlPlane == nil {
		t.Fatalf("unexpected preferred v2 snapshot: %#v", snapshot)
	}
	if source.calls != 1 {
		t.Fatalf("v2-capable peer should require one source build, got %d", source.calls)
	}
}

func TestClientFallsBackToFrozenV1Peer(t *testing.T) {
	legacy := validSnapshot().V1()
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == SnapshotPathV2 {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path != SnapshotPathV1 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(legacy)
	}))
	defer server.Close()

	snapshot, err := NewClient().Fetch(context.Background(), server.URL+SnapshotPathV1, legacy.HostID)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.SchemaVersion != federation.SnapshotSchemaVersion || snapshot.ControlPlane != nil {
		t.Fatalf("unexpected v1 fallback snapshot: %#v", snapshot)
	}
	if strings.Join(paths, ",") != SnapshotPathV2+","+SnapshotPathV1 {
		t.Fatalf("request order = %#v, want v2 then v1", paths)
	}
}

func TestClientRejectsUnexpectedHostIdentity(t *testing.T) {
	source := &snapshotSourceStub{snapshot: validSnapshot()}
	handler, err := NewHandler(source)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	_, err = NewClient().Fetch(context.Background(), server.URL, "host:22222222-2222-4222-9222-222222222222")
	if err == nil || !strings.Contains(err.Error(), "does not match expected") {
		t.Fatalf("expected Host pin mismatch, got %v", err)
	}
}

func TestClientRejectsRemoteCleartextBeforeSendingRequest(t *testing.T) {
	transport := &countingRoundTripper{}
	client, err := NewClientWithHTTPClient(&http.Client{Transport: transport}, DefaultMaxBytes)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Fetch(context.Background(), "http://devbox.example.test/v1/federation/snapshot", "")
	if err == nil || !strings.Contains(err.Error(), "must use HTTPS") {
		t.Fatalf("expected cleartext rejection, got %v", err)
	}
	if transport.calls != 0 {
		t.Fatalf("remote cleartext request was sent %d times", transport.calls)
	}
}

func TestClientRejectsRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, SnapshotPathV1, http.StatusFound)
	}))
	defer server.Close()

	_, err := NewClient().Fetch(context.Background(), server.URL, "")
	if err == nil || !strings.Contains(err.Error(), "redirects are not allowed") {
		t.Fatalf("expected redirect rejection, got %v", err)
	}
}

func TestClientBoundsResponseSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, strings.Repeat("x", 65))
	}))
	defer server.Close()

	client, err := NewClientWithHTTPClient(&http.Client{Timeout: time.Second}, 64)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Fetch(context.Background(), server.URL, "")
	if err == nil || !strings.Contains(err.Error(), "exceeds 64 bytes") {
		t.Fatalf("expected response size rejection, got %v", err)
	}
}

func TestClientRequiresJSONContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, "not-json")
	}))
	defer server.Close()

	_, err := NewClient().Fetch(context.Background(), server.URL, "")
	if err == nil || !strings.Contains(err.Error(), "want application/json") {
		t.Fatalf("expected content type rejection, got %v", err)
	}
}

func TestValidatePeerURLAllowsHTTPSAndLoopbackHTTP(t *testing.T) {
	for _, value := range []string{
		"https://devbox.example.ts.net/v1/federation/snapshot",
		"https://devbox.example.ts.net/v2/federation/snapshot",
		"https://devbox.example.ts.net",
		"http://localhost:7332/v1/federation/snapshot",
		"http://localhost:7332/v2/federation/snapshot",
		"http://127.0.0.1:7332",
		"http://[::1]:7332",
	} {
		if _, err := validatePeerURL(value); err != nil {
			t.Fatalf("validatePeerURL(%q): %v", value, err)
		}
	}
}

func validSnapshot() federation.HostSnapshot {
	return federation.HostSnapshot{
		SchemaVersion: federation.SnapshotSchemaVersionV2,
		HostID:        "host:11111111-1111-4111-9111-111111111111",
		Hostname:      "devbox-01",
		ObservedAt:    time.Date(2026, 8, 23, 20, 0, 0, 0, time.UTC),
		ControlPlane: &controlplane.GetHostControlPlaneResult{
			SchemaVersion: controlplane.SchemaVersion,
			Host:          "devbox-01",
			Attention:     []controlplane.HostAttentionSummary{},
		},
		Instances: []federation.RepositoryInstance{},
	}
}

type countingRoundTripper struct {
	calls int
}

func (r *countingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	r.calls++
	return nil, errors.New("request should not be sent")
}
