package federationhttp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestHandlerReturnsFreshNoStoreSnapshotJSON(t *testing.T) {
	source := &snapshotSourceStub{snapshot: validSnapshot()}
	handler, err := NewHandler(source)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, SnapshotPath, nil)
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
	if snapshot.HostID != source.snapshot.HostID || source.calls != 1 {
		t.Fatalf("unexpected served snapshot=%#v calls=%d", snapshot, source.calls)
	}
}

func TestHandlerRejectsUnsupportedMethodWithoutBuildingSnapshot(t *testing.T) {
	source := &snapshotSourceStub{snapshot: validSnapshot()}
	handler, err := NewHandler(source)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, SnapshotPath, nil))
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
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, SnapshotPath, nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", response.Code)
	}
	if strings.Contains(response.Body.String(), `"schema_version"`) {
		t.Fatalf("partial snapshot leaked on error: %q", response.Body.String())
	}
}

func TestClientFetchesAndPinsLoopbackPeer(t *testing.T) {
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
	if snapshot.HostID != source.snapshot.HostID {
		t.Fatalf("Host ID = %q", snapshot.HostID)
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
		http.Redirect(w, r, SnapshotPath, http.StatusFound)
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
		"https://devbox.example.ts.net",
		"http://localhost:7332/v1/federation/snapshot",
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
		SchemaVersion: federation.SnapshotSchemaVersion,
		HostID:        "host:11111111-1111-4111-9111-111111111111",
		Hostname:      "devbox-01",
		ObservedAt:    time.Date(2026, 8, 23, 20, 0, 0, 0, time.UTC),
		Instances:     []federation.RepositoryInstance{},
	}
}

type countingRoundTripper struct {
	calls int
}

func (r *countingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	r.calls++
	return nil, errors.New("request should not be sent")
}
