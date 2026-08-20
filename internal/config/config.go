package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const FileName = ".specview.yaml"

var defaultConfig = []byte(`version: 1

specs:
  path: specs
  pattern: "*.md"

server:
  host: 127.0.0.1
  port: 7331
`)

type Config struct {
	Version int
	Specs   Specs
	Server  Server
}

type Specs struct {
	Path    string
	Pattern string
}

type Server struct {
	Host string
	Port int
}

func Load(root string) (Config, error) {
	path := filepath.Join(root, FileName)
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	cfg := Config{}
	section := ""
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := len(raw) - len(strings.TrimLeft(raw, " \t"))
		if indent == 0 && strings.HasSuffix(trimmed, ":") {
			section = strings.TrimSuffix(trimmed, ":")
			continue
		}

		key, value, ok := splitKeyValue(trimmed)
		if !ok {
			return Config{}, fmt.Errorf("parse %s line %d: expected key: value", FileName, lineNumber)
		}
		value = unquote(value)

		if indent == 0 {
			section = ""
			switch key {
			case "version":
				n, err := strconv.Atoi(value)
				if err != nil {
					return Config{}, fmt.Errorf("parse %s line %d: invalid version", FileName, lineNumber)
				}
				cfg.Version = n
			default:
				return Config{}, fmt.Errorf("parse %s line %d: unknown key %q", FileName, lineNumber, key)
			}
			continue
		}

		switch section {
		case "specs":
			switch key {
			case "path":
				cfg.Specs.Path = value
			case "pattern":
				cfg.Specs.Pattern = value
			default:
				return Config{}, fmt.Errorf("parse %s line %d: unknown specs key %q", FileName, lineNumber, key)
			}
		case "server":
			switch key {
			case "host":
				cfg.Server.Host = value
			case "port":
				n, err := strconv.Atoi(value)
				if err != nil {
					return Config{}, fmt.Errorf("parse %s line %d: invalid server.port", FileName, lineNumber)
				}
				cfg.Server.Port = n
			default:
				return Config{}, fmt.Errorf("parse %s line %d: unknown server key %q", FileName, lineNumber, key)
			}
		default:
			return Config{}, fmt.Errorf("parse %s line %d: nested key outside a known section", FileName, lineNumber)
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid %s: %w", FileName, err)
	}
	return cfg, nil
}

func splitKeyValue(line string) (string, string, bool) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	return key, value, key != ""
}

func unquote(value string) string {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func (c Config) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported version %d", c.Version)
	}
	if c.Specs.Path == "" {
		return errors.New("specs.path is required")
	}
	if filepath.IsAbs(c.Specs.Path) {
		return errors.New("specs.path must be relative to the repository")
	}
	if c.Specs.Pattern == "" {
		return errors.New("specs.pattern is required")
	}
	if _, err := filepath.Match(c.Specs.Pattern, "example.md"); err != nil {
		return fmt.Errorf("invalid specs.pattern: %w", err)
	}
	if c.Server.Host == "" {
		return errors.New("server.host is required")
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return errors.New("server.port must be between 1 and 65535")
	}
	return nil
}

func Init(root string) (createdConfig bool, createdSpecs bool, err error) {
	configPath := filepath.Join(root, FileName)
	if _, statErr := os.Stat(configPath); errors.Is(statErr, os.ErrNotExist) {
		if writeErr := os.WriteFile(configPath, defaultConfig, 0o644); writeErr != nil {
			return false, false, fmt.Errorf("write %s: %w", FileName, writeErr)
		}
		createdConfig = true
	} else if statErr != nil {
		return false, false, statErr
	}

	specsPath := filepath.Join(root, "specs")
	if _, statErr := os.Stat(specsPath); errors.Is(statErr, os.ErrNotExist) {
		if mkdirErr := os.MkdirAll(specsPath, 0o755); mkdirErr != nil {
			return createdConfig, false, fmt.Errorf("create specs directory: %w", mkdirErr)
		}
		createdSpecs = true
	} else if statErr != nil {
		return createdConfig, false, statErr
	}

	return createdConfig, createdSpecs, nil
}
