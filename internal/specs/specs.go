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
	DependsOn  []string
	Blocks     []string
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

type metadata struct {
	Status    Status
	HasStatus bool
	DependsOn []string
	Blocks    []string
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

	frontMatter, body, metadataErr := splitFrontMatter(data)
	status := StatusNew
	validationError := ""
	parsed := metadata{}
	if metadataErr != nil {
		validationError = metadataErr.Error()
	} else if len(frontMatter) > 0 {
		parsed, err = parseMetadata(frontMatter)
		if err != nil {
			validationError = err.Error()
		} else if parsed.HasStatus {
			status = parsed.Status
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
		DependsOn:  parsed.DependsOn,
		Blocks:     parsed.Blocks,
		ModifiedAt: info.ModTime(),
		Body:       string(body),
		Error:      validationError,
	}, nil
}

func parseMetadata(frontMatter []byte) (metadata, error) {
	result := metadata{}
	lines := strings.Split(string(frontMatter), "\n")
	section := ""
	listKey := ""

	for i, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := len(raw) - len(strings.TrimLeft(raw, " \t"))
		if indent == 0 && strings.HasSuffix(trimmed, ":") {
			section = strings.TrimSuffix(trimmed, ":")
			listKey = ""
			continue
		}
		if indent == 0 {
			section = ""
			listKey = ""
			continue
		}
		if section != "specview" {
			continue
		}

		if strings.HasPrefix(trimmed, "- ") && listKey != "" {
			value := cleanRelationValue(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
			if value == "" {
				return metadata{}, fmt.Errorf("empty %s relation on line %d", listKey, i+1)
			}
			appendRelation(&result, listKey, value)
			continue
		}

		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			return metadata{}, fmt.Errorf("invalid specview metadata on line %d", i+1)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		listKey = ""

		switch key {
		case "status":
			value = strings.Trim(value, "\"'")
			if value == "" {
				return metadata{}, errors.New("specview.status cannot be empty")
			}
			result.Status = Status(value)
			result.HasStatus = true
		case "depends_on", "blocks":
			if value == "" {
				listKey = key
				continue
			}
			for _, relation := range parseRelationList(value) {
				if relation == "" {
					continue
				}
				appendRelation(&result, key, relation)
			}
		}
	}

	result.DependsOn = uniqueRelations(result.DependsOn)
	result.Blocks = uniqueRelations(result.Blocks)
	return result, nil
}

func appendRelation(result *metadata, key, value string) {
	switch key {
	case "depends_on":
		result.DependsOn = append(result.DependsOn, value)
	case "blocks":
		result.Blocks = append(result.Blocks, value)
	}
}

func parseRelationList(value string) []string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		value = strings.TrimSpace(value[1 : len(value)-1])
	}
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if cleaned := cleanRelationValue(part); cleaned != "" {
			out = append(out, cleaned)
		}
	}
	return out
}

func cleanRelationValue(value string) string {
	return strings.Trim(strings.TrimSpace(value), "\"'")
}

func uniqueRelations(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
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
