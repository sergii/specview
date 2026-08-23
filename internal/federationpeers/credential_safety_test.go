package federationpeers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sergii/specview/internal/federationhttp"
)

func TestCredentialErrorRedactionRemovesResolvedSecret(t *testing.T) {
	const secret = "very-secret-token"
	headers := http.Header{"Authorization": []string{secret}}
	err := redactCredentialError(errors.New("request failed with "+secret), headers)
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("credential value leaked from redacted error: %q", err)
	}
	if !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("redaction marker missing: %q", err)
	}
}

func TestMissingCredentialEnvironmentVariableDoesNotSendRequest(t *testing.T) {
	const envName = "SPECVIEW_H22_DEFINITELY_MISSING"
	t.Setenv(envName, "")
	transport := &peerCountingRoundTripper{}
	client, err := federationhttp.NewClientWithHTTPClient(&http.Client{Transport: transport, Timeout: time.Second}, federationhttp.DefaultMaxBytes)
	if err != nil {
		t.Fatal(err)
	}
	peer := testPeer("https://devbox.example.test/v1/federation/snapshot")
	peer.Credentials = &CredentialRef{Type: "env_headers", Headers: map[string]string{"Authorization": envName}}
	store := NewObservationStore(t.TempDir())
	refresher := NewRefresher(client, store)
	refresher.now = func() time.Time { return time.Date(2026, 8, 23, 21, 0, 0, 0, time.UTC) }

	status, err := refresher.Refresh(context.Background(), peer)
	if err == nil {
		t.Fatal("expected missing credential failure")
	}
	if transport.calls != 0 {
		t.Fatalf("request sent %d times before credentials resolved", transport.calls)
	}
	if status.Freshness != FreshnessUnreachable {
		t.Fatalf("freshness = %q, want unreachable", status.Freshness)
	}
	if strings.Contains(status.LastError, "very-secret-token") {
		t.Fatal("secret leaked into observation error")
	}
}

type peerCountingRoundTripper struct {
	calls int
}

func (r *peerCountingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	r.calls++
	return nil, errors.New("request should not be sent")
}
