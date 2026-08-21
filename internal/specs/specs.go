package specs

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const SpecviewAdapterName = "specview"

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

var validStatuses = map[Status]struct{}{
	StatusNew:        {},
	StatusInProgress: {},
	StatusDone:       {},
}

type ArtifactKind string

const (
	ArtifactPolicy      ArtifactKind = "policy"
	ArtifactProposal    ArtifactKind = "proposal"
	ArtifactSpec        ArtifactKind = "spec"
	ArtifactRequirement ArtifactKind = "requirement"
	ArtifactExample     ArtifactKind = "example"
	ArtifactRFC         ArtifactKind = "rfc"
	ArtifactDecision    ArtifactKind = "decision"
	ArtifactDesign      ArtifactKind = "design"
	ArtifactPlan        ArtifactKind = "plan"
	ArtifactTask        ArtifactKind = "task"
	ArtifactContract    ArtifactKind = "contract"
	ArtifactResearch    ArtifactKind = "research"
	ArtifactChecklist   ArtifactKind = "checklist"
)

type ArtifactPlane string

const (
	PlaneKnowledge ArtifactPlane = "knowledge"
	PlaneWork      ArtifactPlane = "work"
)

type ArtifactRole string

const (
	RolePrimary    ArtifactRole = "primary"
	RoleSupporting ArtifactRole = "supporting"
)

type Relation struct {
	Type   string
	Target string
}

type Artifact struct {
	ID         string
	Kind       ArtifactKind
	Plane      ArtifactPlane
	Role       ArtifactRole
	WorkItemID string
	Path       string
	Title      string
	Status     Status
	ModifiedAt time.Time
	Body       string
	Error      string
	Relations  []Relation
}

// IsBoardItem reports whether an artifact represents a unit of active work.
// The legacy fallback keeps pre-normalization spec adapters compatible.
func (a Artifact) IsBoardItem() bool {
	if a.Plane == "" && a.Role == "" {
		return a.Kind == ArtifactSpec
	}
	return a.Plane == PlaneWork && a.Role == RolePrimary
}

// Spec is a compatibility alias for the current spec-oriented UI.
// New domain code should use Artifact.
type Spec = Artifact

type Adapter interface {
	Name() string
	Scan() ([]Artifact, error)
	WatchRoots() []string
}

type SpecviewAdapter struct {
	root    string
	pattern string
}

func NewSpecviewAdapter(root, pattern string) *SpecviewAdapter {
	return &SpecviewAdapter{root: root, pattern: pattern}
}

func (a *SpecviewAdapter) Name() string {
	return SpecviewAdapterName
}

func (a *SpecviewAdapter) Scan() ([]Artifact, error) {
	return scan(a.root, a.pattern)
}

func (a *SpecviewAdapter) WatchRoots() []string {
	return []string{a.root}
}

func NewAdapter(name, root, pattern string) (Adapter, error) {
	cleanRoot := filepath.Clean(root)
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", SpecviewAdapterName:
		return NewSpecviewAdapter(cleanRoot, pattern), nil
	case GitHubSpecKitAdapterName:
		if filepath.Base(cleanRoot) != "specs" {
			return nil, fmt.Errorf("%s expects specs.path to resolve to a top-level specs directory", GitHubSpecKitAdapterName)
		}
		return NewGitHubSpecKitAdapter(filepath.Dir(cleanRoot), cleanRoot), nil
	case OpenSpecAdapterName:
		if filepath.Base(cleanRoot) != "openspec" {
			return nil, fmt.Errorf("%s expects specs.path to resolve to the openspec directory", OpenSpecAdapterName)
		}
		return NewOpenSpecAdapter(cleanRoot), nil
	default:
		return nil, fmt.Errorf("unsupported specs adapter %q", name)
	}
}

type Store struct {
	adapter Adapter
	mu      sync.RWMutex
	items   []Artifact
}

func NewStore(root, pattern string) *Store {
	return NewStoreWithAdapter(NewSpecviewAdapter(root, pattern))
}

func NewStoreWithAdapter(adapter Adapter) *Store {
	return &Store{adapter: adapter}
}

func (s *Store) AdapterName() string {
	return s.adapter.Name()
}

func (s *Store) Refresh() error {
	items, err := s.adapter.Scan()
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.items = items
	s.mu.Unlock()
	return nil
}

func (s *Store) All() []Artifact {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Artifact, len(s.items))
	copy(items, s.items)
	return items
}

func (s *Store) Find(path string) (Artifact, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.items {
		if item.Path == path {
			return item, true
		}
	}
	return Artifact{}, false
}

func scan(root, pattern string) ([]Artifact, error) {
	var items []Artifact
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		matched, err := filepath.Match(pattern, entry.Name())
		if err != nil {
			return err
		}
		if !matched {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		item, err := parseFile(path, filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		items = append(items, item)
		return nil
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan specifications: %w", err)
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	return items, nil
}

func parseFile(fullPath, relPath string) (Artifact, error) {
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return Artifact{}, err
	}
	info, err := os.Stat(fullPath)
	if err != nil {
		return Artifact{}, err
	}

	metadata, body, metadataErr := splitFrontMatter(data)
	status := StatusNew
	validationError := ""
	if metadataErr != nil {
		validationError = metadataErr.Error()
	} else if len(metadata) > 0 {
		parsedStatus, found, err := parseStatus(metadata)
		if err != nil {
			validationError = err.Error()
		} else if found {
			status = parsedStatus
			if _, ok := validStatuses[status]; !ok {
				validationError = fmt.Sprintf("unknown status %q; expected new, in_progress, or done", status)
			}
		}
	}

	title := firstHeading(body)
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
	}

	id := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))

	return Artifact{
		ID:         id,
		Kind:       ArtifactSpec,
		Plane:      PlaneWork,
		Role:       RolePrimary,
		WorkItemID: id,
		Path:       relPath,
		Title:      title,
		Status:     status,
		ModifiedAt: info.ModTime(),
		Body:       string(body),
		Error:      validationError,
		Relations:  nil,
	}, nil
}

func parseStatus(metadata []byte) (Status, bool, error) {
	lines := strings.Split(string(metadata), "\n")
	section := ""
	for i, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " \t"))
		if indent == 0 && strings.HasSuffix(trimmed, ":") {
			section = strings.TrimSuffix(trimmed, ":")
			continue
		}
		if indent == 0 {
			section = ""
			continue
		}
		if section != "specview" {
			continue
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			return "", false, fmt.Errorf("invalid specview metadata on line %d", i+1)
		}
		if strings.TrimSpace(parts[0]) == "status" {
			value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			if value == "" {
				return "", false, errors.New("specview.status cannot be empty")
			}
			return Status(value), true, nil
		}
	}
	return StatusNew, false, nil
}

func splitFrontMatter(data []byte) (metadata, body []byte, err error) {
	normalized := bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	if !bytes.HasPrefix(normalized, []byte("---\n")) {
		return nil, normalized, nil
	}

	rest := normalized[4:]
	end := bytes.Index(rest, []byte("\n---\n"))
	if end < 0 {
		return nil, normalized, errors.New("front matter starts with --- but has no closing ---")
	}
	return rest[:end], rest[end+5:], nil
}

func firstHeading(body []byte) string {
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}
