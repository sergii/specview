package main

import "testing"

func TestParseFederationPullArgs(t *testing.T) {
	const hostID = "host:11111111-1111-4111-9111-111111111111"

	tests := []struct {
		name     string
		args     []string
		wantURL  string
		wantHost string
		wantErr  bool
	}{
		{
			name:    "url only",
			args:    []string{"https://devbox.example.ts.net"},
			wantURL: "https://devbox.example.ts.net",
		},
		{
			name:     "expected host separate",
			args:     []string{"--expect-host", hostID, "https://devbox.example.ts.net"},
			wantURL:  "https://devbox.example.ts.net",
			wantHost: hostID,
		},
		{
			name:     "expected host inline",
			args:     []string{"https://devbox.example.ts.net", "--expect-host=" + hostID},
			wantURL:  "https://devbox.example.ts.net",
			wantHost: hostID,
		},
		{name: "missing url", args: nil, wantErr: true},
		{name: "missing expected host value", args: []string{"--expect-host"}, wantErr: true},
		{name: "empty expected host value", args: []string{"--expect-host=", "https://devbox.example.ts.net"}, wantErr: true},
		{name: "duplicate expected host", args: []string{"--expect-host=" + hostID, "--expect-host", hostID, "https://devbox.example.ts.net"}, wantErr: true},
		{name: "unknown option", args: []string{"--insecure", "https://devbox.example.ts.net"}, wantErr: true},
		{name: "multiple urls", args: []string{"https://one.example.test", "https://two.example.test"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotHost, err := parseFederationPullArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got url=%q host=%q", gotURL, gotHost)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if gotURL != tt.wantURL {
				t.Fatalf("url = %q, want %q", gotURL, tt.wantURL)
			}
			if gotHost != tt.wantHost {
				t.Fatalf("expected Host ID = %q, want %q", gotHost, tt.wantHost)
			}
		})
	}
}

func TestRunFederationServeRejectsArgumentsBeforeOpeningListener(t *testing.T) {
	if err := runFederation([]string{"serve", "0.0.0.0:7332"}); err == nil {
		t.Fatal("federation serve must not accept a bind address")
	}
}
