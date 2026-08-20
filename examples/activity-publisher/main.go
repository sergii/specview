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
	"syscall"
	"time"
)

const activityDir = ".specview/runtime/activity"

type record struct {
	Version     int       `json:"version"`
	SessionID   string    `json:"session_id"`
	Agent       agent     `json:"agent"`
	Spec        string    `json:"spec"`
	State       string    `json:"state"`
	StartedAt   time.Time `json:"started_at"`
	HeartbeatAt time.Time `json:"heartbeat_at"`
}

type agent struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func main() {
	var root, spec, agentID, agentLabel string
	flag.StringVar(&root, "root", ".", "project root")
	flag.StringVar(&spec, "spec", "", "project-relative specification path")
	flag.StringVar(&agentID, "agent-id", "agent", "machine-readable agent identifier")
	flag.StringVar(&agentLabel, "agent-label", "Agent", "human-readable agent label")
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

	startedAt := time.Now().UTC()
	publish := func() error {
		now := time.Now().UTC()
		value := record{
			Version:     1,
			SessionID:   sessionID,
			Agent:       agent{ID: agentID, Label: agentLabel},
			Spec:        filepath.ToSlash(filepath.Clean(spec)),
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

func newSessionID() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
