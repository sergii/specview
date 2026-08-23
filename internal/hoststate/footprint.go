package hoststate

import (
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const repositoryFootprintTTL = 5 * time.Minute

// RepositoryFootprint is a cached filesystem projection. It is deliberately
// derived state: Git/filesystem remain the source of truth.
type RepositoryFootprint struct {
	WorkingTreeBytes int64
	GitBytes         int64
	TotalBytes       int64
	MeasuredAt       time.Time
}

func (f RepositoryFootprint) Available() bool {
	return !f.MeasuredAt.IsZero()
}

func (f RepositoryFootprint) Label() string {
	if !f.Available() {
		return ""
	}
	return formatFootprintBytes(f.TotalBytes)
}

func (f RepositoryFootprint) BreakdownLabel() string {
	if !f.Available() {
		return ""
	}
	return fmt.Sprintf("Working tree %s · Git %s", formatFootprintBytes(f.WorkingTreeBytes), formatFootprintBytes(f.GitBytes))
}

type footprintCacheEntry struct {
	footprint RepositoryFootprint
	expiresAt time.Time
}

type repositoryFootprintCache struct {
	mu      sync.Mutex
	entries map[string]footprintCacheEntry
}

var defaultRepositoryFootprints = repositoryFootprintCache{entries: make(map[string]footprintCacheEntry)}

// Footprint returns a best-effort cached disk-usage projection for a repository.
// Measurement errors intentionally degrade to an unavailable value so UI reads
// can never make repository pages fail.
func (r Repository) Footprint() RepositoryFootprint {
	return defaultRepositoryFootprints.measure(r.Root, time.Now())
}

func (c *repositoryFootprintCache) measure(root string, now time.Time) RepositoryFootprint {
	identity := normalizeFilesystemPath(root)

	c.mu.Lock()
	if cached, ok := c.entries[identity]; ok && now.Before(cached.expiresAt) {
		c.mu.Unlock()
		return cached.footprint
	}
	c.mu.Unlock()

	footprint, err := measureRepositoryFootprint(root, now)
	if err != nil {
		footprint = RepositoryFootprint{}
	}

	c.mu.Lock()
	c.entries[identity] = footprintCacheEntry{footprint: footprint, expiresAt: now.Add(repositoryFootprintTTL)}
	c.mu.Unlock()
	return footprint
}

func measureRepositoryFootprint(root string, measuredAt time.Time) (RepositoryFootprint, error) {
	root = filepath.Clean(root)
	workingBytes, err := walkFootprint(root, filepath.Join(root, ".git"))
	if err != nil {
		return RepositoryFootprint{}, err
	}

	gitDir, err := gitCommonDir(root)
	if err != nil {
		return RepositoryFootprint{}, err
	}
	gitBytes, err := walkFootprint(gitDir, "")
	if err != nil {
		return RepositoryFootprint{}, err
	}

	return RepositoryFootprint{
		WorkingTreeBytes: workingBytes,
		GitBytes:         gitBytes,
		TotalBytes:       workingBytes + gitBytes,
		MeasuredAt:       measuredAt,
	}, nil
}

func gitCommonDir(root string) (string, error) {
	command := exec.Command("git", "-C", root, "rev-parse", "--path-format=absolute", "--git-common-dir")
	output, err := command.Output()
	if err == nil {
		return filepath.Clean(strings.TrimSpace(string(output))), nil
	}

	output, err = exec.Command("git", "-C", root, "rev-parse", "--git-common-dir").Output()
	if err != nil {
		return "", fmt.Errorf("resolve Git common directory: %w", err)
	}
	candidate := strings.TrimSpace(string(output))
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(root, candidate)
	}
	return filepath.Clean(candidate), nil
}

func walkFootprint(root, skipPath string) (int64, error) {
	root = filepath.Clean(root)
	if skipPath != "" {
		skipPath = filepath.Clean(skipPath)
	}

	var total int64
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if skipPath != "" && filepath.Clean(path) == skipPath {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() || entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}

func formatFootprintBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}

	units := []string{"KB", "MB", "GB", "TB"}
	value := float64(bytes)
	for _, unit := range units {
		value /= 1024
		if value < 1024 || unit == "TB" {
			if value < 10 {
				formatted := fmt.Sprintf("%.1f", value)
				formatted = strings.TrimSuffix(formatted, ".0")
				return formatted + " " + unit
			}
			return fmt.Sprintf("%.0f %s", value, unit)
		}
	}
	return fmt.Sprintf("%d B", bytes)
}
