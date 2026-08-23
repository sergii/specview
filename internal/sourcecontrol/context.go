package sourcecontrol

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// RepositoryContext combines local Git state with optional remote provider
// context. Git is authoritative for local repository/worktree state; provider
// context is an independently degradable projection.
type RepositoryContext struct {
	Git      GitContext
	Provider ProviderContext
}

type GitContext struct {
	Remote    string
	Worktrees []Worktree
}

type Worktree struct {
	Path       string
	Branch     string
	Head       string
	Detached   bool
	DirtyCount int
	Upstream   string
	Ahead      int
	Behind     int
	LastCommit string
}

func (w Worktree) BranchLabel() string {
	if w.Detached {
		return "detached"
	}
	if w.Branch == "" {
		return "unknown"
	}
	return w.Branch
}

func (w Worktree) ShortHead() string {
	if len(w.Head) <= 8 {
		return w.Head
	}
	return w.Head[:8]
}

func (w Worktree) ChangeLabel() string {
	switch w.DirtyCount {
	case 0:
		return "clean"
	case 1:
		return "1 change"
	default:
		return fmt.Sprintf("%d changes", w.DirtyCount)
	}
}

func (w Worktree) SyncLabel() string {
	if w.Detached {
		return "detached"
	}
	if w.Upstream == "" {
		return "no upstream"
	}
	if w.Ahead == 0 && w.Behind == 0 {
		return w.Upstream + " · synced"
	}
	parts := make([]string, 0, 2)
	if w.Ahead > 0 {
		parts = append(parts, fmt.Sprintf("↑%d", w.Ahead))
	}
	if w.Behind > 0 {
		parts = append(parts, fmt.Sprintf("↓%d", w.Behind))
	}
	return w.Upstream + " · " + strings.Join(parts, " ")
}

type ProviderContext struct {
	Name         string
	Matched      bool
	Available    bool
	Repository   string
	WebURL       string
	Error        string
	PullRequests []PullRequest
}

type PullRequest struct {
	Number int
	Title  string
	URL    string
	State  string
	Draft  bool
	Base   string
	Head   string
	Checks CheckSummary
}

func (p PullRequest) StateLabel() string {
	if p.Draft {
		return "draft"
	}
	if p.State == "" {
		return "unknown"
	}
	return strings.ToLower(p.State)
}

type CheckSummary struct {
	Total   int
	Passed  int
	Failed  int
	Pending int
	Skipped int
}

func (c CheckSummary) Label() string {
	if c.Total == 0 {
		return "no checks"
	}
	parts := make([]string, 0, 4)
	if c.Failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", c.Failed))
	}
	if c.Pending > 0 {
		parts = append(parts, fmt.Sprintf("%d pending", c.Pending))
	}
	if c.Passed > 0 {
		parts = append(parts, fmt.Sprintf("%d passed", c.Passed))
	}
	if c.Skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", c.Skipped))
	}
	return strings.Join(parts, " · ")
}

// Source is the portable repository-context boundary consumed by the web
// projection. Implementations may use local Git plus one or more remote forge
// adapters, but consumers do not depend on those mechanics.
type Source interface {
	Inspect(context.Context, string) (RepositoryContext, error)
}

type ProviderAdapter interface {
	Name() string
	Supports(remote string) bool
	Inspect(context.Context, GitContext) ProviderContext
}

type providerCacheEntry struct {
	observedAt time.Time
	context    ProviderContext
}

type Service struct {
	providers   []ProviderAdapter
	providerTTL time.Duration
	now         func() time.Time
	mu          sync.Mutex
	cache       map[string]providerCacheEntry
}

func NewService(providers ...ProviderAdapter) *Service {
	return NewServiceWithProviderTTL(30*time.Second, providers...)
}

func NewServiceWithProviderTTL(ttl time.Duration, providers ...ProviderAdapter) *Service {
	return &Service{
		providers:   append([]ProviderAdapter(nil), providers...),
		providerTTL: ttl,
		now:         time.Now,
		cache:       make(map[string]providerCacheEntry),
	}
}

func DefaultService() *Service {
	return NewService(NewGitHubCLIAdapter())
}

func (s *Service) Inspect(ctx context.Context, root string) (RepositoryContext, error) {
	gitContext, err := InspectGit(root)
	if err != nil {
		return RepositoryContext{}, err
	}
	result := RepositoryContext{Git: gitContext}

	for _, provider := range s.providers {
		if !provider.Supports(gitContext.Remote) {
			continue
		}
		key := provider.Name() + "\x00" + gitContext.Remote
		if cached, ok := s.cachedProvider(key); ok {
			result.Provider = cached
			return result, nil
		}
		providerContext := provider.Inspect(ctx, gitContext)
		if providerContext.Name == "" {
			providerContext.Name = provider.Name()
		}
		providerContext.Matched = true
		s.storeProvider(key, providerContext)
		result.Provider = providerContext
		return result, nil
	}

	return result, nil
}

func (s *Service) cachedProvider(key string) (ProviderContext, bool) {
	if s.providerTTL <= 0 {
		return ProviderContext{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.cache[key]
	if !ok || s.now().Sub(entry.observedAt) >= s.providerTTL {
		return ProviderContext{}, false
	}
	return entry.context, true
}

func (s *Service) storeProvider(key string, providerContext ProviderContext) {
	if s.providerTTL <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[key] = providerCacheEntry{observedAt: s.now(), context: providerContext}
}
