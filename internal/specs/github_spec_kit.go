package specs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const GitHubSpecKitAdapterName = "github-spec-kit"

type GitHubSpecKitAdapter struct {
	projectRoot string
	specRoot    string
}

func NewGitHubSpecKitAdapter(projectRoot, specRoot string) *GitHubSpecKitAdapter {
	return &GitHubSpecKitAdapter{projectRoot: projectRoot, specRoot: specRoot}
}

func (a *GitHubSpecKitAdapter) Name() string {
	return GitHubSpecKitAdapterName
}

func (a *GitHubSpecKitAdapter) WatchRoots() []string {
	return []string{
		a.specRoot,
		filepath.Join(a.projectRoot, ".specify", "memory"),
	}
}

func (a *GitHubSpecKitAdapter) Scan() ([]Artifact, error) {
	var artifacts []Artifact

	constitutionPath := filepath.Join(a.projectRoot, ".specify", "memory", "constitution.md")
	if artifact, ok, err := readOptionalArtifact(constitutionPath, ".specify/memory/constitution.md", "constitution", ArtifactPolicy, StatusDone, nil); err != nil {
		return nil, err
	} else if ok {
		artifacts = append(artifacts, artifact)
	}

	entries, err := os.ReadDir(a.specRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return artifacts, nil
		}
		return nil, fmt.Errorf("scan GitHub Spec Kit specs: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		featureArtifacts, err := a.scanFeature(entry.Name())
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, featureArtifacts...)
	}

	sort.Slice(artifacts, func(i, j int) bool {
		if artifacts[i].Kind == artifacts[j].Kind {
			return artifacts[i].Path < artifacts[j].Path
		}
		if artifacts[i].Kind == ArtifactSpec {
			return true
		}
		if artifacts[j].Kind == ArtifactSpec {
			return false
		}
		return artifacts[i].Path < artifacts[j].Path
	})
	return artifacts, nil
}

func (a *GitHubSpecKitAdapter) scanFeature(featureID string) ([]Artifact, error) {
	featureRoot := filepath.Join(a.specRoot, featureID)
	specPath := filepath.Join(featureRoot, "spec.md")
	if _, err := os.Stat(specPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	status, err := deriveSpecKitStatus(featureRoot)
	if err != nil {
		return nil, err
	}

	main, _, err := readOptionalArtifact(
		specPath,
		filepath.ToSlash(filepath.Join(featureID, "spec.md")),
		featureID,
		ArtifactSpec,
		status,
		nil,
	)
	if err != nil {
		return nil, err
	}

	latestModified := main.ModifiedAt
	artifacts := []Artifact{main}

	knownFiles := []struct {
		name string
		kind ArtifactKind
	}{
		{name: "plan.md", kind: ArtifactPlan},
		{name: "research.md", kind: ArtifactResearch},
		{name: "data-model.md", kind: ArtifactDesign},
		{name: "quickstart.md", kind: ArtifactChecklist},
		{name: "tasks.md", kind: ArtifactTask},
	}

	for _, known := range knownFiles {
		path := filepath.Join(featureRoot, known.name)
		rel := filepath.ToSlash(filepath.Join(featureID, known.name))
		artifact, ok, err := readOptionalArtifact(
			path,
			rel,
			featureID+":"+strings.TrimSuffix(known.name, filepath.Ext(known.name)),
			known.kind,
			status,
			[]Relation{{Type: "belongs_to", Target: featureID}},
		)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		artifacts = append(artifacts, artifact)
		if artifact.ModifiedAt.After(latestModified) {
			latestModified = artifact.ModifiedAt
		}
	}

	contractRoot := filepath.Join(featureRoot, "contracts")
	err = filepath.WalkDir(contractRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relWithinFeature, err := filepath.Rel(featureRoot, path)
		if err != nil {
			return err
		}
		artifact, ok, err := readOptionalArtifact(
			path,
			filepath.ToSlash(filepath.Join(featureID, relWithinFeature)),
			featureID+":contract:"+filepath.ToSlash(relWithinFeature),
			ArtifactContract,
			status,
			[]Relation{{Type: "belongs_to", Target: featureID}},
		)
		if err != nil {
			return err
		}
		if ok {
			artifacts = append(artifacts, artifact)
			if artifact.ModifiedAt.After(latestModified) {
				latestModified = artifact.ModifiedAt
			}
		}
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	// The dashboard card represents the feature, not only spec.md. Show the
	// latest activity time from its known artifacts while preserving spec.md
	// as the feature's source document.
	artifacts[0].ModifiedAt = latestModified
	return artifacts, nil
}

func deriveSpecKitStatus(featureRoot string) (Status, error) {
	tasksPath := filepath.Join(featureRoot, "tasks.md")
	data, err := os.ReadFile(tasksPath)
	if err == nil {
		total, completed := countMarkdownTasks(string(data))
		if total > 0 && completed == total {
			return StatusDone, nil
		}
		return StatusInProgress, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return StatusNew, err
	}

	if _, err := os.Stat(filepath.Join(featureRoot, "plan.md")); err == nil {
		return StatusInProgress, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return StatusNew, err
	}

	return StatusNew, nil
}

func countMarkdownTasks(body string) (total, completed int) {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) < 5 || !strings.HasPrefix(trimmed, "- [") {
			continue
		}
		switch trimmed[3] {
		case ' ', 'x', 'X':
			total++
			if trimmed[3] == 'x' || trimmed[3] == 'X' {
				completed++
			}
		}
	}
	return total, completed
}

func readOptionalArtifact(fullPath, relPath, id string, kind ArtifactKind, status Status, relations []Relation) (Artifact, bool, error) {
	data, err := os.ReadFile(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Artifact{}, false, nil
		}
		return Artifact{}, false, err
	}
	info, err := os.Stat(fullPath)
	if err != nil {
		return Artifact{}, false, err
	}

	_, body, metadataErr := splitFrontMatter(data)
	validationError := ""
	if metadataErr != nil {
		validationError = metadataErr.Error()
		body = data
	}
	title := firstHeading(body)
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
	}

	return Artifact{
		ID:         id,
		Kind:       kind,
		Path:       relPath,
		Title:      title,
		Status:     status,
		ModifiedAt: info.ModTime(),
		Body:       string(body),
		Error:      validationError,
		Relations:  relations,
	}, true, nil
}

var _ Adapter = (*GitHubSpecKitAdapter)(nil)
