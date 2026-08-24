package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sergii/specview/internal/hoststate"
)

func TestHostPageExposesSharedHostNavigation(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)
	recorder := httptest.NewRecorder()
	server.index(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}

	body := recorder.Body.String()
	for _, want := range []string{
		`aria-label="Host views"`,
		`href="/" data-host-nav="/"`,
		`href="/history" data-host-nav="/history"`,
		`href="/federation" data-host-nav="/federation"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("host page missing shared navigation %q", want)
		}
	}
}
