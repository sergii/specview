package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"path/filepath"
	"strings"
)

type CorrelationOutcome string

const (
	CorrelationMatch     CorrelationOutcome = "match"
	CorrelationAmbiguous CorrelationOutcome = "ambiguous"
	CorrelationDistinct  CorrelationOutcome = "distinct"
	CorrelationConflict  CorrelationOutcome = "conflict"
)

type RepositoryFingerprint struct {
	ExplicitID      string `json:"explicit_id,omitempty"`
	Name            string `json:"name,omitempty"`
	GitRemote       string `json:"git_remote,omitempty"`
	ForgeProvider   string `json:"forge_provider,omitempty"`
	ForgeRepository string `json:"forge_repository,omitempty"`
}

type CorrelationResult struct {
	Outcome CorrelationOutcome `json:"outcome"`
	Reasons []string           `json:"reasons,omitempty"`
}

func RepositoryInstanceID(hostID, root string) (string, error) {
	if !validHostID(strings.TrimSpace(hostID)) {
		return "", errors.New("valid host identity is required")
	}
	root = normalizeLocalRoot(root)
	if root == "" || root == "." || root == "/" {
		return "", errors.New("repository root is required")
	}
	sum := sha256.Sum256([]byte(hostID + "\x00" + root))
	return "instance:" + hex.EncodeToString(sum[:16]), nil
}

func CorrelateRepositories(left, right RepositoryFingerprint) CorrelationResult {
	left = normalizeFingerprint(left)
	right = normalizeFingerprint(right)

	explicitSame := left.ExplicitID != "" && right.ExplicitID != "" && left.ExplicitID == right.ExplicitID
	explicitDifferent := left.ExplicitID != "" && right.ExplicitID != "" && left.ExplicitID != right.ExplicitID
	remoteSame := left.GitRemote != "" && right.GitRemote != "" && left.GitRemote == right.GitRemote
	remoteDifferent := left.GitRemote != "" && right.GitRemote != "" && left.GitRemote != right.GitRemote
	leftForge := forgeIdentity(left)
	rightForge := forgeIdentity(right)
	forgeSame := leftForge != "" && rightForge != "" && leftForge == rightForge
	forgeDifferent := leftForge != "" && rightForge != "" && leftForge != rightForge

	if explicitDifferent {
		return CorrelationResult{Outcome: CorrelationDistinct, Reasons: []string{"explicit_project_id_differs"}}
	}
	if explicitSame {
		if remoteDifferent || forgeDifferent {
			return CorrelationResult{Outcome: CorrelationConflict, Reasons: contradictionReasons(remoteDifferent, forgeDifferent)}
		}
		return CorrelationResult{Outcome: CorrelationMatch, Reasons: []string{"explicit_project_id_matches"}}
	}

	if left.Name != "" && right.Name != "" && left.Name != right.Name {
		return CorrelationResult{Outcome: CorrelationDistinct, Reasons: []string{"repository_name_differs"}}
	}
	if remoteDifferent || forgeDifferent {
		return CorrelationResult{Outcome: CorrelationDistinct, Reasons: contradictionReasons(remoteDifferent, forgeDifferent)}
	}
	if left.Name == "" || right.Name == "" {
		return CorrelationResult{Outcome: CorrelationAmbiguous, Reasons: []string{"repository_name_missing"}}
	}
	if remoteSame || forgeSame {
		reasons := []string{"repository_name_matches"}
		if remoteSame {
			reasons = append(reasons, "git_remote_matches")
		}
		if forgeSame {
			reasons = append(reasons, "forge_identity_matches")
		}
		return CorrelationResult{Outcome: CorrelationMatch, Reasons: reasons}
	}
	return CorrelationResult{Outcome: CorrelationAmbiguous, Reasons: []string{"repository_name_matches_without_corroboration"}}
}

func NormalizeRepositoryName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "/")
	for strings.Contains(value, "//") {
		value = strings.ReplaceAll(value, "//", "/")
	}
	value = strings.Trim(value, "/")
	value = strings.ToLower(value)
	value = strings.TrimSuffix(value, ".git")
	return value
}

func NormalizeGitRemote(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if strings.Contains(value, "://") {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Host == "" {
			return ""
		}
		host := strings.ToLower(parsed.Host)
		path := NormalizeRepositoryName(parsed.Path)
		if path == "" {
			return ""
		}
		return host + "/" + path
	}

	if at := strings.LastIndex(value, "@"); at >= 0 {
		remainder := value[at+1:]
		if colon := strings.Index(remainder, ":"); colon > 0 {
			host := strings.ToLower(strings.TrimSpace(remainder[:colon]))
			path := NormalizeRepositoryName(remainder[colon+1:])
			if host != "" && path != "" {
				return host + "/" + path
			}
		}
	}
	return ""
}

func normalizeFingerprint(value RepositoryFingerprint) RepositoryFingerprint {
	value.ExplicitID = strings.TrimSpace(value.ExplicitID)
	value.Name = NormalizeRepositoryName(value.Name)
	value.GitRemote = NormalizeGitRemote(value.GitRemote)
	value.ForgeProvider = strings.ToLower(strings.TrimSpace(value.ForgeProvider))
	value.ForgeRepository = NormalizeRepositoryName(value.ForgeRepository)
	return value
}

func forgeIdentity(value RepositoryFingerprint) string {
	if value.ForgeProvider == "" || value.ForgeRepository == "" {
		return ""
	}
	return value.ForgeProvider + ":" + value.ForgeRepository
}

func contradictionReasons(remoteDifferent, forgeDifferent bool) []string {
	var reasons []string
	if remoteDifferent {
		reasons = append(reasons, "git_remote_differs")
	}
	if forgeDifferent {
		reasons = append(reasons, "forge_identity_differs")
	}
	return reasons
}

func normalizeLocalRoot(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(root))
}
