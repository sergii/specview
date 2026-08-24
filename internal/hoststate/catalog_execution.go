package hoststate

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sergii/specview/internal/config"
)

// ObserveExecutions persists normalized logical execution sessions. This is the
// canonical product path; process IDs are diagnostics and never session keys.
func (c *Catalog) ObserveExecutions(executions []ExecutionSession, now time.Time) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	activeIDs := make(map[string]struct{})
	seenIDs := make(map[string]string)
	changed := false
	materialChanged := false
	heartbeatChanged := false

	for _, raw := range executions {
		execution, err := normalizeExecutionForCatalog(raw)
		if err != nil {
			return false, err
		}
		if previousRoot, exists := seenIDs[execution.ID]; exists && previousRoot != execution.RepositoryRoot {
			return false, fmt.Errorf("logical execution session %q observed in multiple repositories", execution.ID)
		}
		seenIDs[execution.ID] = execution.RepositoryRoot

		repo, repoChanged, err := c.ensureExecutionRepositoryLocked(execution.RepositoryRoot, now)
		if err != nil {
			return false, err
		}
		if repoChanged {
			changed = true
			materialChanged = true
		}
		activeIDs[logicalSessionKey(repo.ID, execution.ID)] = struct{}{}

		existing := findSessionByID(repo.Sessions, execution.ID)
		if existing >= 0 {
			session := &repo.Sessions[existing]
			if applyLogicalExecution(session, execution, now) {
				changed = true
				materialChanged = true
			}
			if !session.LastSeenAt.Equal(now) {
				session.LastSeenAt = now
				changed = true
				heartbeatChanged = true
			}
		} else {
			legacy, kept := collapseEligibleLegacyFragments(repo.Sessions, execution)
			repo.Sessions = kept
			startedAt := execution.StartedAt
			if startedAt.IsZero() {
				startedAt = now
			}
			for _, fragment := range legacy {
				if !fragment.StartedAt.IsZero() && (startedAt.IsZero() || fragment.StartedAt.Before(startedAt)) {
					startedAt = fragment.StartedAt
				}
			}
			repo.Sessions = append(repo.Sessions, Session{
				ID:           execution.ID,
				IdentityKind: SessionIdentityLogical,
				Adapter:      execution.Adapter,
				Agent:        execution.Agent,
				ProcessIDs:   append([]int(nil), execution.ProcessIDs...),
				CWD:          execution.CWD,
				WorktreeRoot: execution.WorktreeRoot,
				StartedAt:    startedAt,
				LastSeenAt:   now,
				Active:       true,
			})
			changed = true
			materialChanged = true
		}

		if repo.LastSeenAt.Before(now) {
			repo.LastSeenAt = now
			changed = true
			heartbeatChanged = true
		}
	}

	for repoID, repo := range c.repos {
		for i := range repo.Sessions {
			session := &repo.Sessions[i]
			if !session.Active {
				continue
			}
			if session.IdentityKind == SessionIdentityLogical {
				if _, ok := activeIDs[logicalSessionKey(repoID, session.ID)]; ok {
					continue
				}
			}
			ended := now
			session.Active = false
			session.EndedAt = &ended
			changed = true
			materialChanged = true
		}
	}

	return c.finishObservationLocked(changed, materialChanged, heartbeatChanged, now)
}

// ensureExecutionRepositoryLocked preserves the durable repository record when
// upgrading a legacy catalog. Historical catalog IDs are not rewritten merely
// because the current derived root ID differs; the filesystem root is the
// compatibility bridge. Multiple records for one root fail closed instead of
// allowing a live logical session to attach nondeterministically.
func (c *Catalog) ensureExecutionRepositoryLocked(root string, now time.Time) (*Repository, bool, error) {
	repoID := repositoryID(root)
	if _, ok := c.repos[repoID]; ok {
		repo, changed := c.ensureRepositoryLocked(root, now)
		return repo, changed, nil
	}

	var matched *Repository
	for _, candidate := range c.repos {
		if !sameFilesystemPath(candidate.Root, root) {
			continue
		}
		if matched != nil && matched.ID != candidate.ID {
			return nil, false, fmt.Errorf("multiple catalog repositories share filesystem root %q: %s and %s", root, matched.ID, candidate.ID)
		}
		matched = candidate
	}
	if matched == nil {
		repo, changed := c.ensureRepositoryLocked(root, now)
		return repo, changed, nil
	}

	changed := false
	convention, err := c.detect(root)
	if err != nil {
		if matched.DetectionError != err.Error() || matched.Convention != (config.Convention{}) {
			matched.DetectionError = err.Error()
			matched.Convention = config.Convention{}
			changed = true
		}
	} else if matched.Convention != convention || matched.DetectionError != "" {
		matched.Convention = convention
		matched.DetectionError = ""
		changed = true
	}
	return matched, changed, nil
}

func normalizeExecutionForCatalog(execution ExecutionSession) (ExecutionSession, error) {
	execution.ID = strings.TrimSpace(execution.ID)
	execution.Adapter = strings.TrimSpace(execution.Adapter)
	execution.Agent = strings.TrimSpace(execution.Agent)
	execution.RepositoryRoot = strings.TrimSpace(execution.RepositoryRoot)
	if execution.ID == "" {
		return ExecutionSession{}, fmt.Errorf("logical execution session id is required")
	}
	if execution.Adapter == "" {
		return ExecutionSession{}, fmt.Errorf("logical execution session %q adapter is required", execution.ID)
	}
	if execution.Agent == "" {
		return ExecutionSession{}, fmt.Errorf("logical execution session %q agent is required", execution.ID)
	}
	if execution.RepositoryRoot == "" {
		return ExecutionSession{}, fmt.Errorf("logical execution session %q repository root is required", execution.ID)
	}
	execution.RepositoryRoot = filepath.Clean(execution.RepositoryRoot)
	if execution.CWD != "" {
		execution.CWD = filepath.Clean(execution.CWD)
	}
	if execution.WorktreeRoot != "" {
		execution.WorktreeRoot = filepath.Clean(execution.WorktreeRoot)
	}
	execution.ProcessIDs = normalizeProcessIDs(execution.ProcessIDs)
	return execution, nil
}

func logicalSessionKey(repositoryID, sessionID string) string {
	return repositoryID + "\x00" + sessionID
}

func findSessionByID(sessions []Session, id string) int {
	for i := range sessions {
		if sessions[i].ID == id {
			return i
		}
	}
	return -1
}

func applyLogicalExecution(session *Session, execution ExecutionSession, now time.Time) bool {
	changed := false
	if session.IdentityKind != SessionIdentityLogical {
		session.IdentityKind = SessionIdentityLogical
		changed = true
	}
	if session.Adapter != execution.Adapter {
		session.Adapter = execution.Adapter
		changed = true
	}
	if session.Agent != execution.Agent {
		session.Agent = execution.Agent
		changed = true
	}
	if !equalIntSlices(session.ProcessIDs, execution.ProcessIDs) {
		session.ProcessIDs = append([]int(nil), execution.ProcessIDs...)
		changed = true
	}
	if session.CWD != execution.CWD {
		session.CWD = execution.CWD
		changed = true
	}
	if session.WorktreeRoot != execution.WorktreeRoot {
		session.WorktreeRoot = execution.WorktreeRoot
		changed = true
	}
	if session.StartedAt.IsZero() {
		if execution.StartedAt.IsZero() {
			session.StartedAt = now
		} else {
			session.StartedAt = execution.StartedAt
		}
		changed = true
	} else if !execution.StartedAt.IsZero() && execution.StartedAt.Before(session.StartedAt) {
		session.StartedAt = execution.StartedAt
		changed = true
	}
	if !session.Active || session.EndedAt != nil {
		session.Active = true
		session.EndedAt = nil
		changed = true
	}
	return changed
}

func collapseEligibleLegacyFragments(sessions []Session, execution ExecutionSession) ([]Session, []Session) {
	legacy := make([]Session, 0)
	kept := make([]Session, 0, len(sessions))
	for _, session := range sessions {
		if legacyFragmentMatches(session, execution) {
			legacy = append(legacy, session)
			continue
		}
		kept = append(kept, session)
	}
	return legacy, kept
}

func legacyFragmentMatches(session Session, execution ExecutionSession) bool {
	if !session.Active || session.IdentityKind != SessionIdentityLegacyPID {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(session.Agent), strings.TrimSpace(execution.Agent)) {
		return false
	}
	if !adapterCompatible(session.Adapter, execution.Adapter) {
		return false
	}
	return processIDsOverlap(session.ProcessIDs, execution.ProcessIDs)
}

func adapterCompatible(legacyAdapter, logicalAdapter string) bool {
	legacyAdapter = strings.TrimSpace(legacyAdapter)
	logicalAdapter = strings.TrimSpace(logicalAdapter)
	return legacyAdapter == "legacy" || strings.EqualFold(legacyAdapter, logicalAdapter)
}

func processIDsOverlap(left, right []int) bool {
	if len(left) == 0 || len(right) == 0 {
		return false
	}
	set := make(map[int]struct{}, len(left))
	for _, pid := range left {
		set[pid] = struct{}{}
	}
	for _, pid := range right {
		if _, ok := set[pid]; ok {
			return true
		}
	}
	return false
}

func equalIntSlices(left, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func sortedSessionIDs(sessions []Session) []string {
	ids := make([]string, 0, len(sessions))
	for _, session := range sessions {
		ids = append(ids, session.ID)
	}
	sort.Strings(ids)
	return ids
}
