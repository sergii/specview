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

const OpenSpecAdapterName = "openspec"

type OpenSpecAdapter struct {
	root string
}

func NewOpenSpecAdapter(root string) *OpenSpecAdapter {
	return &OpenSpecAdapter{root: filepath.Clean(root)}
}

func (a *OpenSpecAdapter) Name() string {
	return OpenSpecAdapterName
}

func (a *OpenSpecAdapter) WatchRoots() []string {
	// OpenSpec owns this namespace. Watching it is still narrow compared with
	// observing the whole source tree and also catches schema/config changes.
	return []string{a.root}
}

func (a *OpenSpecAdapter) Scan() ([]Artifact, error) {
	current, err := a.scanCurrentSpecs()
	if err != nil {
		return nil, err
	}
	changes, err := a.scanActiveChanges()
	if err != nil {
		return nil, err
	}

	artifacts := append(current, changes...)
	sort.Slice(artifacts, func(i, j int) bool {
		if artifacts[i].Plane != artifacts[j].Plane {
			return artifacts[i].Plane < artifacts[j].Plane
		}
		if artifacts[i].Role != artifacts[j].Role {
			return artifacts[i].Role < artifacts[j].Role
		}
		return artifacts[i].Path < artifacts[j].Path
	})
	return artifacts, nil
}

func (a *OpenSpecAdapter) scanCurrentSpecs() ([]Artifact, error) {
	root := filepath.Join(a.root, "specs")
	var artifacts []Artifact
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != "spec.md" {
			return nil
		}

		capabilityPath, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		capability := filepath.ToSlash(capabilityPath)
		rel, err := filepath.Rel(a.root, path)
		if err != nil {
			return err
		}

		artifact, ok, err := readOptionalArtifact(
			path,
			filepath.ToSlash(rel),
			"current:"+capability,
			ArtifactSpec,
			StatusDone,
			nil,
		)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		artifact.Plane = PlaneKnowledge
		artifact.Role = RolePrimary
		artifact.Relations = []Relation{{Type: "defines", Target: "capability:" + capability}}
		artifacts = append(artifacts, artifact)
		return nil
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan OpenSpec current specs: %w", err)
	}
	return artifacts, nil
}

func (a *OpenSpecAdapter) scanActiveChanges() ([]Artifact, error) {
	root := filepath.Join(a.root, "changes")
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan OpenSpec changes: %w", err)
	}

	var artifacts []Artifact
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "archive" {
			continue
		}
		changeArtifacts, err := a.scanChange(entry.Name())
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, changeArtifacts...)
	}
	return artifacts, nil
}

func (a *OpenSpecAdapter) scanChange(changeID string) ([]Artifact, error) {
	root := filepath.Join(a.root, "changes", changeID)
	status, err := deriveOpenSpecStatus(root)
	if err != nil {
		return nil, err
	}

	var artifacts []Artifact
	latestModified := int64(0)
	primaryIndex := -1

	known := []struct {
		name string
		kind ArtifactKind
	}{
		{name: "proposal.md", kind: ArtifactProposal},
		{name: "design.md", kind: ArtifactDesign},
		{name: "tasks.md", kind: ArtifactTask},
	}

	for _, item := range known {
		fullPath := filepath.Join(root, item.name)
		rel, err := filepath.Rel(a.root, fullPath)
		if err != nil {
			return nil, err
		}
		artifact, ok, err := readOptionalArtifact(
			fullPath,
			filepath.ToSlash(rel),
			changeID+":"+strings.TrimSuffix(item.name, filepath.Ext(item.name)),
			item.kind,
			status,
			[]Relation{{Type: "belongs_to", Target: changeID}},
		)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		artifact.Plane = PlaneWork
		artifact.Role = RoleSupporting
		artifact.WorkItemID = changeID
		if item.name == "proposal.md" {
			artifact.ID = changeID
			artifact.Role = RolePrimary
			primaryIndex = len(artifacts)
		}
		if artifact.ModifiedAt.UnixNano() > latestModified {
			latestModified = artifact.ModifiedAt.UnixNano()
		}
		artifacts = append(artifacts, artifact)
	}

	deltaRoot := filepath.Join(root, "specs")
	err = filepath.WalkDir(deltaRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != "spec.md" {
			return nil
		}
		capabilityPath, err := filepath.Rel(deltaRoot, filepath.Dir(path))
		if err != nil {
			return err
		}
		capability := filepath.ToSlash(capabilityPath)
		rel, err := filepath.Rel(a.root, path)
		if err != nil {
			return err
		}
		artifact, ok, err := readOptionalArtifact(
			path,
			filepath.ToSlash(rel),
			changeID+":delta:"+capability,
			ArtifactSpec,
			status,
			[]Relation{
				{Type: "belongs_to", Target: changeID},
				{Type: "changes", Target: "current:" + capability},
			},
		)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		artifact.Plane = PlaneWork
		artifact.Role = RoleSupporting
		artifact.WorkItemID = changeID
		if artifact.ModifiedAt.UnixNano() > latestModified {
			latestModified = artifact.ModifiedAt.UnixNano()
		}
		artifacts = append(artifacts, artifact)
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("scan OpenSpec change %s delta specs: %w", changeID, err)
	}

	// OpenSpec is intentionally fluid and permits artifacts to be created in
	// different orders. If proposal.md does not exist yet, promote the first
	// available artifact so the change still has one normalized work-item root.
	if primaryIndex < 0 && len(artifacts) > 0 {
		primaryIndex = 0
		artifacts[0].ID = changeID
		artifacts[0].Role = RolePrimary
	}
	if primaryIndex >= 0 && latestModified > 0 {
		for i := range artifacts {
			if i == primaryIndex {
				for _, candidate := range artifacts {
					if candidate.ModifiedAt.After(artifacts[i].ModifiedAt) {
						artifacts[i].ModifiedAt = candidate.ModifiedAt
					}
				}
				break
			}
		}
	}

	return artifacts, nil
}

func deriveOpenSpecStatus(changeRoot string) (Status, error) {
	tasksPath := filepath.Join(changeRoot, "tasks.md")
	if data, err := os.ReadFile(tasksPath); err == nil {
		total, completed := countMarkdownTasks(string(data))
		if total > 0 && total == completed {
			return StatusDone, nil
		}
		return StatusInProgress, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return StatusNew, err
	}

	for _, path := range []string{
		filepath.Join(changeRoot, "design.md"),
		filepath.Join(changeRoot, "specs"),
	} {
		if _, err := os.Stat(path); err == nil {
			return StatusInProgress, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return StatusNew, err
		}
	}
	return StatusNew, nil
}

var _ Adapter = (*OpenSpecAdapter)(nil)
