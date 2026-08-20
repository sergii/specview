package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const activityDir = ".specview/runtime/activity"

type record struct {
	Version     int       `json:"version"`
	SessionID   string    `json:"session_id"`
	Agent       agent     `json:"agent"`
	Spec        string    `json:"spec"`
	Files       []string  `json:"files,omitempty"`
	State       string    `json:"state"`
	StartedAt   time.Time `json:"started_at"`
	HeartbeatAt time.Time `json:"heartbeat_at"`
}

type agent struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func main() {
	var root, spec, agentID, agentLabel, filesValue string
	flag.StringVar(&root, "root", ".", "project root")
	flag.StringVar(&spec, "spec", "", "project-relative specification path")
	flag.StringVar(&agentID, "agent-id", "agent", "machine-readable agent identifier")
	flag.StringVar(&agentLabel, "agent-label", "Agent", "human-readable agent label")
	flag.StringVar(&filesValue, "files", "", "comma-separated project-relative files currently touched by the agent")
	flag.Parse()
	if spec == "" {
		fmt.Fprintln(os.Stderr, "activity-publisher: --spec is required")
		os.Exit(2)
	}

	sessionID, err := newSessionID()
	if err != nil {
		fmt.Fprintf(os.Stderr, "activity-publisher: generate session id: %v\n", err)
		os.Exit(1)
	}

	dir := filepath.Join(root, activityDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "activity-publisher: create runtime directory: %v\n", err)
		os.Exit(1)
	}
	path := filepath.Join(dir, sessionID+".json")
	defer os.Remove(path)

	files := parseFiles(filesValue)
	startedAt := time.Now().UTC()
	publish := func() error {
		now := time.Now().UTC()
		value := record{
			Version:     1,
			SessionID:   sessionID,
			Agent:       agent{ID: agentID, Label: agentLabel},
			Spec:        filepath.ToSlash(filepath.Clean(spec)),
			Files:       files,
			State:       "working",
			StartedAt:   startedAt,
			HeartbeatAt: now,
		}
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		data = append(data, '\n')
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, data, 0o644); err != nil {
			return err
		}
		return os.Rename(tmp, path)
	}

	if err := publish(); err != nil {
		fmt.Fprintf(os.Stderr, "activity-publisher: publish: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("publishing %s on %s as %s (%s)\n", agentLabel, spec, sessionID, path)
	if len(files) > 0 {
		fmt.Printf("touching %s\n", strings.Join(files, ", "))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("activity publisher stopped")
			return
		case <-ticker.C:
			if err := publish(); err != nil {
				fmt.Fprintf(os.Stderr, "activity-publisher: heartbeat: %v\n", err)
			}
		}
	}
}

func parseFiles(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		part = filepath.ToSlash(filepath.Clean(part))
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	return out
}

func newSessionID() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
