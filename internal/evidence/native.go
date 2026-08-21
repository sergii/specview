package evidence

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type NativeAdapter struct {
	root string
}

func NewNativeAdapter(root string) *NativeAdapter {
	return &NativeAdapter{root: filepath.Clean(root)}
}

func (a *NativeAdapter) Name() string {
	return NativeAdapterName
}

func (a *NativeAdapter) WatchRoots() []string {
	return []string{a.root}
}

func (a *NativeAdapter) Scan() ([]Record, error) {
	var records []Record
	err := filepath.WalkDir(a.root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			return nil
		}

		rel, err := filepath.Rel(a.root, path)
		if err != nil {
			return err
		}
		record, parseErr := parseNativeFile(path)
		record.Path = filepath.ToSlash(rel)
		if parseErr != nil {
			record.Error = parseErr.Error()
		}
		records = append(records, record)
		return nil
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan evidence: %w", err)
	}

	sort.Slice(records, func(i, j int) bool { return records[i].Path < records[j].Path })
	return records, nil
}

func parseNativeFile(path string) (Record, error) {
	file, err := os.Open(path)
	if err != nil {
		return Record{}, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	var record Record
	if err := decoder.Decode(&record); err != nil {
		return record, fmt.Errorf("decode evidence: %w", err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return record, fmt.Errorf("decode evidence: multiple JSON values")
		}
		return record, fmt.Errorf("decode evidence: %w", err)
	}

	if err := record.Validate(); err != nil {
		return record, fmt.Errorf("validate evidence: %w", err)
	}
	return record, nil
}

var _ Adapter = (*NativeAdapter)(nil)
