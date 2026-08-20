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

type Spec struct {
	Path       string
	Title      string
	Status     Status
	ModifiedAt time.Time
	Body       string
	Error      string
}

type Store struct {
	root    string
	pattern string
	mu      sync.RWMutex
	items   []Spec
}

func NewStore(root, pattern string) *Store {
	return &Store{root: root, pattern: pattern}
}

func (s *Store) Refresh() error {
	items, err := scan(s.root, s.pattern)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.items = items
	s.mu.Unlock()
	return nil
}

func (s *Store) All() []Spec {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Spec, len(s.items))
	copy(items, s.items)
	return items
}

func (s *Store) Find(path string) (Spec, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.items {
		if item.Path == path {
			return item, true
		}
	}
	return Spec{}, false
}

func scan(root, pattern string) ([]Spec, error) {
	var items []Spec
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

func parseFile(fullPath, relPath string) (Spec, error) {
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return Spec{}, err
	}
	info, err := os.Stat(fullPath)
	if err != nil {
		return Spec{}, err
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

	return Spec{
		Path:       relPath,
		Title:      title,
		Status:     status,
		ModifiedAt: info.ModTime(),
		Body:       string(body),
		Error:      validationError,
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
