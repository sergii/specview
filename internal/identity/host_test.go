package identity

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHostIdentityPersistsAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "host.json")
	createdAt := time.Date(2026, 8, 23, 19, 0, 0, 0, time.UTC)
	first, err := loadOrCreateHost(path, func() time.Time { return createdAt }, bytes.NewReader(bytes.Repeat([]byte{0x11}, 16)))
	if err != nil {
		t.Fatal(err)
	}
	if first.Version != HostIdentityVersion {
		t.Fatalf("version = %d", first.Version)
	}
	if first.ID != "host:11111111-1111-4111-9111-111111111111" {
		t.Fatalf("id = %q", first.ID)
	}
	if !first.CreatedAt.Equal(createdAt) {
		t.Fatalf("created_at = %s", first.CreatedAt)
	}

	second, err := loadOrCreateHost(path, func() time.Time { return createdAt.Add(time.Hour) }, bytes.NewReader(bytes.Repeat([]byte{0x22}, 16)))
	if err != nil {
		t.Fatal(err)
	}
	if second != first {
		t.Fatalf("identity changed across reopen: first=%#v second=%#v", first, second)
	}
}

func TestHostIdentityRejectsCorruptOrUnsupportedState(t *testing.T) {
	for name, body := range map[string]string{
		"invalid-json":        `not-json`,
		"unknown-field":       `{"version":1,"id":"host:11111111-1111-4111-9111-111111111111","created_at":"2026-08-23T19:00:00Z","extra":true}`,
		"unsupported-version": `{"version":2,"id":"host:11111111-1111-4111-9111-111111111111","created_at":"2026-08-23T19:00:00Z"}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "host.json")
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := LoadOrCreateHost(path)
			if err == nil {
				t.Fatal("expected invalid persisted identity to fail")
			}
		})
	}
}

func TestHostIdentityValidation(t *testing.T) {
	valid := HostIdentity{
		Version:   HostIdentityVersion,
		ID:        "host:550e8400-e29b-41d4-a716-446655440000",
		CreatedAt: time.Date(2026, 8, 23, 19, 0, 0, 0, time.UTC),
	}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}

	invalid := valid
	invalid.ID = strings.Replace(valid.ID, "host:", "machine:", 1)
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected invalid host prefix to fail")
	}
}
