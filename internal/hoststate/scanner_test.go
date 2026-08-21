package hoststate

import "testing"

func TestLooksLikeCodexCommandForms(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{name: "native binary", command: "/opt/homebrew/bin/codex", want: true},
		{name: "native target binary", command: "/tmp/codex-aarch64-apple-darwin", want: true},
		{name: "node package path", command: "node /opt/homebrew/lib/node_modules/@openai/codex/bin/codex.js", want: true},
		{name: "node shim", command: "node /opt/homebrew/bin/codex", want: true},
		{name: "shell shim", command: "/bin/sh /Users/test/.local/bin/codex", want: true},
		{name: "npx", command: "npx codex", want: true},
		{name: "ordinary node", command: "node server.js", want: false},
		{name: "grep should not count", command: "grep codex", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikeCodex(tt.command); got != tt.want {
				t.Fatalf("looksLikeCodex(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}
