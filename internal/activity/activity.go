package activity

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	RelativeDir = ".specview/runtime/activity"
	DefaultTTL  = 30 * time.Second
)

type Agent struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type Record struct {
	Version     int       `json:"version"`
	SessionID   string    `json:"session_id"`
	Agent       Agent     `json:"agent"`
	Spec        string    `json:"spec"`
	State       string    `json:"state"`
	StartedAt   time.Time `json:"started_at"`
	HeartbeatAt time.Time `json:"heartbeat_at"`
}

type ParseError struct {
	Path    string
	Message string
}

type Store struct {
	root    string
	mu      sync.RWMutex
	records []Record
	errors  []ParseError
}

func NewStore(root string) *Store { return &Store{root: root} }

func (s *Store) Refresh() error {
	var records []Record
	var parseErrors []ParseError

	err := filepath.WalkDir(s.root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var record Record
		if err := json.Unmarshal(data, &record); err != nil {
			parseErrors = append(parseErrors, ParseError{Path: path, Message: err.Error()})
			return nil
		}
		if err := validate(record); err != nil {
			parseErrors = append(parseErrors, ParseError{Path: path, Message: err.Error()})
			return nil
		}
		record.Spec = filepath.ToSlash(filepath.Clean(record.Spec))
		record.Agent.ID = strings.TrimSpace(record.Agent.ID)
		record.Agent.Label = strings.TrimSpace(record.Agent.Label)
		records = append(records, record)
		return nil
	})
	if os.IsNotExist(err) {
		err = nil
	}
	if err != nil {
		return fmt.Errorf("scan agent activity: %w", err)
	}

	sort.Slice(records, func(i, j int) bool {
		if records[i].Spec != records[j].Spec {
			return records[i].Spec < records[j].Spec
		}
		return records[i].SessionID < records[j].SessionID
	})

	s.mu.Lock()
	s.records = records
	s.errors = parseErrors
	s.mu.Unlock()
	return nil
}

func (s *Store) ActiveFor(spec string, now time.Time) []Record {
	spec = filepath.ToSlash(filepath.Clean(spec))
	s.mu.RLock()
	defer s.mu.RUnlock()

	active := make([]Record, 0)
	for _, record := range s.records {
		if record.Spec != spec || record.State != "working" || !fresh(record, now) {
			continue
		}
		active = append(active, record)
	}
	return active
}

func (s *Store) Errors() []ParseError {
	s.mu.RLock()
	defer s.mu.RUnlock()
	errs := make([]ParseError, len(s.errors))
	copy(errs, s.errors)
	return errs
}

func AgentLabel(record Record) string {
	if record.Agent.Label != "" {
		return record.Agent.Label
	}
	if record.Agent.ID != "" {
		return record.Agent.ID
	}
	return "Agent"
}

func ExpiresAt(record Record) time.Time { return record.HeartbeatAt.Add(DefaultTTL) }

func fresh(record Record, now time.Time) bool {
	if record.HeartbeatAt.IsZero() {
		return false
	}
	age := now.Sub(record.HeartbeatAt)
	return age >= -5*time.Second && age <= DefaultTTL
}

func validate(record Record) error {
	if record.Version != 1 {
		return fmt.Errorf("unsupported activity version %d", record.Version)
	}
	if strings.TrimSpace(record.SessionID) == "" {
		return fmt.Errorf("session_id is required")
	}
	if strings.TrimSpace(record.Spec) == "" {
		return fmt.Errorf("spec is required")
	}
	if strings.TrimSpace(record.State) == "" {
		return fmt.Errorf("state is required")
	}
	if record.HeartbeatAt.IsZero() {
		return fmt.Errorf("heartbeat_at is required")
	}
	return nil
}
