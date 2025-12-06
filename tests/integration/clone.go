//go:build integration

package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

const cloneCompleteMarker = ".clone_complete"

var unsafePathChars = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

// CloneResult contains the result of a clone operation.
type CloneResult struct {
	FromCache bool
	Path      string
}

// CloneRepo performs a shallow clone of a repository to the cache directory.
// If the repository already exists in cache, it returns the cached path.
func CloneRepo(repo Repository) (*CloneResult, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return nil, err
	}
	repoDir := filepath.Join(cacheDir, sanitizeDirName(repo.Name, repo.Ref))
	completionMarker := filepath.Join(repoDir, cloneCompleteMarker)

	// Check if already fully cloned
	if _, err := os.Stat(completionMarker); err == nil {
		return &CloneResult{
			FromCache: true,
			Path:      repoDir,
		}, nil
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	// Clean up any partial clone
	_ = os.RemoveAll(repoDir)

	if err := shallowClone(repo.URL, repo.Ref, repoDir); err != nil {
		_ = os.RemoveAll(repoDir)
		return nil, fmt.Errorf("shallow clone %s: %w", repo.Name, err)
	}

	// Mark as complete
	if err := os.WriteFile(completionMarker, []byte{}, 0644); err != nil {
		return nil, fmt.Errorf("create completion marker: %w", err)
	}

	return &CloneResult{
		FromCache: false,
		Path:      repoDir,
	}, nil
}

func shallowClone(url, ref, dest string) error {
	args := []string{
		"clone",
		"--depth=1",
		"--branch=" + ref,
		"--single-branch",
		url,
		dest,
	}

	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w (output: %s)", err, string(output))
	}

	return nil
}

func getCacheDir() (string, error) {
	testDataDir, err := getTestDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(testDataDir, "cache"), nil
}

func sanitizeDirName(name, ref string) string {
	safeName := unsafePathChars.ReplaceAllString(name, "_")
	safeRef := unsafePathChars.ReplaceAllString(ref, "_")
	return fmt.Sprintf("%s-%s", safeName, safeRef)
}
