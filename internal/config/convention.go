package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	AdapterSpecview      = "specview"
	AdapterGitHubSpecKit = "github-spec-kit"
	AdapterOpenSpec      = "openspec"
	AdapterKiro          = "kiro"
	AdapterBMAD          = "bmad"
)

type Convention struct {
	Adapter    string `json:"adapter"`
	Label      string `json:"label"`
	Path       string `json:"path"`
	Recognized bool   `json:"recognized"`
	Supported  bool   `json:"supported"`
}

func DetectConvention(root string) (Convention, error) {
	if _, err := os.Stat(filepath.Join(root, FileName)); err == nil {
		cfg, loadErr := Load(root)
		if loadErr != nil {
			return Convention{}, loadErr
		}
		return convention(cfg.Specs.Adapter, cfg.Specs.Path), nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return Convention{}, err
	}

	type candidate struct {
		matched bool
		value   Convention
	}
	candidates := []candidate{
		{matched: isDir(filepath.Join(root, ".specify")), value: convention(AdapterGitHubSpecKit, "specs")},
		{matched: isOpenSpecRoot(filepath.Join(root, "openspec")), value: convention(AdapterOpenSpec, "openspec")},
		{matched: isDir(filepath.Join(root, ".kiro", "specs")), value: convention(AdapterKiro, filepath.Join(".kiro", "specs"))},
		{matched: isDir(filepath.Join(root, "_bmad-output")), value: convention(AdapterBMAD, "_bmad-output")},
	}

	var matches []Convention
	for _, candidate := range candidates {
		if candidate.matched {
			matches = append(matches, candidate.value)
		}
	}
	if len(matches) > 1 {
		return Convention{}, fmt.Errorf("multiple specification conventions detected; configure %s explicitly", FileName)
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	return Convention{}, nil
}

func convention(adapter, path string) Convention {
	c := Convention{
		Adapter:    adapter,
		Label:      ConventionLabel(adapter),
		Path:       filepath.ToSlash(path),
		Recognized: true,
	}
	switch adapter {
	case AdapterSpecview, AdapterGitHubSpecKit, AdapterOpenSpec:
		c.Supported = true
	}
	return c
}

func ConventionLabel(adapter string) string {
	switch adapter {
	case AdapterSpecview:
		return "Specview"
	case AdapterGitHubSpecKit:
		return "GitHub Spec Kit"
	case AdapterOpenSpec:
		return "OpenSpec"
	case AdapterKiro:
		return "Kiro"
	case AdapterBMAD:
		return "BMAD"
	default:
		if adapter == "" {
			return ""
		}
		return adapter
	}
}
