package projectstate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sergii/specview/internal/acceptance"
	"github.com/sergii/specview/internal/config"
	"github.com/sergii/specview/internal/evidence"
	"github.com/sergii/specview/internal/revision"
	"github.com/sergii/specview/internal/sourcecontrol"
	"github.com/sergii/specview/internal/specs"
)

type Project struct {
	RepositoryRoot string
	ProjectRoot    string
	Convention     config.Convention
	SpecPath       string
	Pattern        string
	Policy         acceptance.Policy
}

type AcceptanceResult struct {
	Decision          acceptance.Decision
	Revision          revision.Resolution
	EvidenceCount     int
	EvaluationPending bool
}

func Resolve(repositoryRoot string) (Project, error) {
	repositoryRoot = filepath.Clean(repositoryRoot)
	convention, err := config.DetectConvention(repositoryRoot)
	if err != nil {
		return Project{}, err
	}

	project := Project{
		RepositoryRoot: repositoryRoot,
		ProjectRoot:    repositoryRoot,
		Convention:     convention,
		SpecPath:       convention.Path,
		Pattern:        "*.md",
	}

	if _, err := os.Stat(filepath.Join(repositoryRoot, config.FileName)); err == nil {
		cfg, loadErr := config.Load(repositoryRoot)
		if loadErr != nil {
			return Project{}, loadErr
		}
		project.ProjectRoot = cfg.ResolveProjectRoot(repositoryRoot)
		project.SpecPath = cfg.Specs.Path
		project.Pattern = cfg.Specs.Pattern
		project.Policy = cfg.AcceptancePolicy()
		project.Convention = config.Convention{
			Adapter:    cfg.Specs.Adapter,
			Label:      config.ConventionLabel(cfg.Specs.Adapter),
			Path:       filepath.ToSlash(cfg.Specs.Path),
			Recognized: true,
			Supported:  supportedAdapter(cfg.Specs.Adapter),
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Project{}, err
	}

	return project, nil
}

func (p Project) WorkItems() ([]specs.Artifact, error) {
	if !p.Convention.Recognized {
		return nil, errors.New("no recognized specification pattern")
	}
	if !p.Convention.Supported {
		return nil, fmt.Errorf("unsupported specification adapter %q", p.Convention.Adapter)
	}

	adapter, err := specs.NewAdapter(
		p.Convention.Adapter,
		filepath.Join(p.ProjectRoot, p.SpecPath),
		p.Pattern,
	)
	if err != nil {
		return nil, err
	}
	store := specs.NewStoreWithAdapter(adapter)
	if err := store.Refresh(); err != nil {
		return nil, err
	}

	items := make([]specs.Artifact, 0)
	for _, artifact := range store.All() {
		if artifact.IsBoardItem() {
			items = append(items, artifact)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].WorkItemID == items[j].WorkItemID {
			return items[i].Path < items[j].Path
		}
		return items[i].WorkItemID < items[j].WorkItemID
	})
	return items, nil
}

func (p Project) FindWorkItem(workItemID string) (specs.Artifact, error) {
	workItemID = strings.TrimSpace(workItemID)
	if workItemID == "" {
		return specs.Artifact{}, errors.New("work item id is required")
	}
	items, err := p.WorkItems()
	if err != nil {
		return specs.Artifact{}, err
	}
	for _, item := range items {
		if item.WorkItemID == workItemID {
			return item, nil
		}
	}
	return specs.Artifact{}, fmt.Errorf("work item %q not found", workItemID)
}

func (p Project) Evidence(workItemID string) ([]evidence.Record, error) {
	workItemID = strings.TrimSpace(workItemID)
	if workItemID == "" {
		return nil, errors.New("work item id is required")
	}

	store := evidence.NewStore(evidence.NewNativeAdapter(filepath.Join(p.ProjectRoot, ".specview", "evidence")))
	if err := store.Refresh(); err != nil {
		return nil, err
	}
	records := store.ForWorkItem(workItemID)
	sort.Slice(records, func(i, j int) bool {
		if records[i].ObservedAt.Equal(records[j].ObservedAt) {
			return records[i].ID > records[j].ID
		}
		return records[i].ObservedAt.After(records[j].ObservedAt)
	})
	return records, nil
}

func (p Project) EvaluateAcceptance(workItemID string, git sourcecontrol.GitContext) (AcceptanceResult, error) {
	workItemID = strings.TrimSpace(workItemID)
	if workItemID == "" {
		return AcceptanceResult{}, errors.New("work item id is required")
	}

	if len(p.Policy.Required) == 0 {
		decision, err := acceptance.Evaluate(p.Policy, "", "", nil)
		if err != nil {
			return AcceptanceResult{}, err
		}
		return AcceptanceResult{Decision: decision}, nil
	}

	resolution := revision.ResolveGit(p.ProjectRoot, git)
	records, err := p.Evidence(workItemID)
	if err != nil {
		return AcceptanceResult{}, err
	}
	result := AcceptanceResult{
		Revision:      resolution,
		EvidenceCount: len(records),
	}
	if !resolution.Available {
		result.Decision = acceptance.Decision{
			WorkItemID: workItemID,
			State:      acceptance.StateWaiting,
		}
		result.EvaluationPending = true
		return result, nil
	}

	decision, err := acceptance.Evaluate(p.Policy, workItemID, resolution.Revision, records)
	if err != nil {
		return AcceptanceResult{}, err
	}
	result.Decision = decision
	return result, nil
}

func supportedAdapter(adapter string) bool {
	switch adapter {
	case config.AdapterSpecview, config.AdapterGitHubSpecKit, config.AdapterOpenSpec:
		return true
	default:
		return false
	}
}
