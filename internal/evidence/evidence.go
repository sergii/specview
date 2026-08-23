package evidence

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const NativeAdapterName = "native"

type Kind string

const (
	KindCompile        Kind = "compile"
	KindTypecheck      Kind = "typecheck"
	KindStaticAnalysis Kind = "static_analysis"
	KindLint           Kind = "lint"
	KindTest           Kind = "test"
	KindAcceptance     Kind = "acceptance"
	KindContract       Kind = "contract"
	KindProperty       Kind = "property"
	KindFuzz           Kind = "fuzz"
	KindMutation       Kind = "mutation"
	KindArchitecture   Kind = "architecture"
	KindSecurity       Kind = "security"
	KindMigration      Kind = "migration"
	KindPerformance    Kind = "performance"
	KindHardware       Kind = "hardware"
	KindReview         Kind = "review"
	KindCI             Kind = "ci"
	KindCustom         Kind = "custom"
)

var validKinds = map[Kind]struct{}{
	KindCompile:        {},
	KindTypecheck:      {},
	KindStaticAnalysis: {},
	KindLint:           {},
	KindTest:           {},
	KindAcceptance:     {},
	KindContract:       {},
	KindProperty:       {},
	KindFuzz:           {},
	KindMutation:       {},
	KindArchitecture:   {},
	KindSecurity:       {},
	KindMigration:      {},
	KindPerformance:    {},
	KindHardware:       {},
	KindReview:         {},
	KindCI:             {},
	KindCustom:         {},
}

type Result string

const (
	ResultQueued  Result = "queued"
	ResultRunning Result = "running"
	ResultPassed  Result = "passed"
	ResultFailed  Result = "failed"
	ResultError   Result = "error"
	ResultSkipped Result = "skipped"
)

var validResults = map[Result]struct{}{
	ResultQueued:  {},
	ResultRunning: {},
	ResultPassed:  {},
	ResultFailed:  {},
	ResultError:   {},
	ResultSkipped: {},
}

type Record struct {
	Version    int                `json:"version"`
	ID         string             `json:"id"`
	WorkItemID string             `json:"work_item_id"`
	Revision   string             `json:"revision"`
	Check      string             `json:"check"`
	Kind       Kind               `json:"kind"`
	Provider   string             `json:"provider"`
	Result     Result             `json:"result"`
	StartedAt  *time.Time         `json:"started_at,omitempty"`
	FinishedAt *time.Time         `json:"finished_at,omitempty"`
	ObservedAt time.Time          `json:"observed_at"`
	Summary    string             `json:"summary,omitempty"`
	Metrics    map[string]float64 `json:"metrics,omitempty"`

	Path  string `json:"-"`
	Error string `json:"-"`
}

func (r Record) Terminal() bool {
	switch r.Result {
	case ResultPassed, ResultFailed, ResultError, ResultSkipped:
		return true
	default:
		return false
	}
}

func (r Record) MatchesRevision(revision string) bool {
	return r.Error == "" && revision != "" && r.Revision == revision
}

func (r Record) PassedForRevision(revision string) bool {
	return r.MatchesRevision(revision) && r.Result == ResultPassed
}

func (r Record) Validate() error {
	if r.Version != 1 {
		return fmt.Errorf("unsupported evidence version %d", r.Version)
	}
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(r.WorkItemID) == "" {
		return fmt.Errorf("work_item_id is required")
	}
	if strings.TrimSpace(r.Revision) == "" {
		return fmt.Errorf("revision is required")
	}
	if strings.TrimSpace(r.Check) == "" {
		return fmt.Errorf("check is required")
	}
	if _, ok := validKinds[r.Kind]; !ok {
		return fmt.Errorf("unknown evidence kind %q", r.Kind)
	}
	if strings.TrimSpace(r.Provider) == "" {
		return fmt.Errorf("provider is required")
	}
	if _, ok := validResults[r.Result]; !ok {
		return fmt.Errorf("unknown evidence result %q", r.Result)
	}
	if r.ObservedAt.IsZero() {
		return fmt.Errorf("observed_at is required")
	}
	if r.Terminal() && r.FinishedAt == nil {
		return fmt.Errorf("finished_at is required for terminal result %q", r.Result)
	}
	if r.StartedAt != nil && r.FinishedAt != nil && r.FinishedAt.Before(*r.StartedAt) {
		return fmt.Errorf("finished_at cannot be before started_at")
	}
	return nil
}

type Adapter interface {
	Name() string
	Scan() ([]Record, error)
	WatchRoots() []string
}

type Store struct {
	adapter Adapter
	mu      sync.RWMutex
	records []Record
}

func NewStore(adapter Adapter) *Store {
	return &Store{adapter: adapter}
}

func (s *Store) AdapterName() string {
	return s.adapter.Name()
}

func (s *Store) Refresh() error {
	records, err := s.adapter.Scan()
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.records = records
	s.mu.Unlock()
	return nil
}

func (s *Store) All() []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	records := make([]Record, len(s.records))
	copy(records, s.records)
	return records
}

func (s *Store) ForWorkItem(workItemID string) []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var records []Record
	for _, record := range s.records {
		if record.WorkItemID == workItemID {
			records = append(records, record)
		}
	}
	return records
}
