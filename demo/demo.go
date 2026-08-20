package demo

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const configContent = `version: 1

project:
  name: "Demo Project"
  demo: true

specs:
  path: specs
  pattern: "*.md"

server:
  host: 127.0.0.1
  port: 7331
`

//go:embed specs/*.md
var files embed.FS

func Create(root string) (int, error) {
	configPath := filepath.Join(root, ".specview.yaml")
	if _, err := os.Stat(configPath); err == nil {
		return 0, errors.New(".specview.yaml already exists; demo initialization requires an uninitialized repository")
	} else if !errors.Is(err, os.ErrNotExist) {
		return 0, err
	}

	entries, err := fs.ReadDir(files, "specs")
	if err != nil {
		return 0, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(root, "specs", entry.Name())
		if _, err := os.Stat(path); err == nil {
			return 0, fmt.Errorf("demo file %s already exists", filepath.ToSlash(filepath.Join("specs", entry.Name())))
		} else if !errors.Is(err, os.ErrNotExist) {
			return 0, err
		}
	}

	if err := os.MkdirAll(filepath.Join(root, "specs"), 0o755); err != nil {
		return 0, err
	}
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		return 0, err
	}

	created := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := files.ReadFile(filepath.ToSlash(filepath.Join("specs", entry.Name())))
		if err != nil {
			return created, err
		}
		path := filepath.Join(root, "specs", entry.Name())
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return created, err
		}
		created++
	}
	return created, nil
}
