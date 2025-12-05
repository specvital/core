// Package config provides scope-based configuration file resolution.
package config

import (
	"os"
	"path/filepath"
	"regexp"
)

const (
	defaultMaxDepth = 20

	// Project root indicator files
	fileGitDir      = ".git"
	fileGoMod       = "go.mod"
	filePackageJSON = "package.json"

	// Monorepo indicator files
	fileLernaJSON         = "lerna.json"
	fileNxJSON            = "nx.json"
	filePnpmWorkspaceYAML = "pnpm-workspace.yaml"
	fileRushJSON          = "rush.json"
)

type Resolver struct {
	cache    *Cache
	maxDepth int
}

func NewResolver(cache *Cache, maxDepth int) *Resolver {
	if maxDepth <= 0 {
		maxDepth = defaultMaxDepth
	}
	return &Resolver{
		cache:    cache,
		maxDepth: maxDepth,
	}
}

var (
	projectRootIndicators = []string{fileGoMod, fileGitDir, filePackageJSON}
	monorepoIndicators    = []string{filePnpmWorkspaceYAML, fileLernaJSON, fileRushJSON, fileNxJSON}
	workspacesPattern     = regexp.MustCompile(`"workspaces"\s*:`)
)

// ResolveConfig finds the nearest config file by traversing up directories.
// Stops at project root boundary (go.mod, .git, package.json) + one level for monorepo configs.
func (r *Resolver) ResolveConfig(filePath string, patterns []string) (string, bool) {
	dir := filepath.Dir(filePath)
	var foundProjectRoot bool
	var visitedDirs []string

	for depth := 0; depth < r.maxDepth; depth++ {
		if r.cache != nil {
			if cached, found := r.cache.Get(dir, patterns); found {
				return cached, cached != ""
			}
		}

		visitedDirs = append(visitedDirs, dir)

		if configPath, found := r.findConfigInDir(dir, patterns); found {
			return configPath, true
		}

		if !foundProjectRoot && isProjectRoot(dir) {
			foundProjectRoot = true
		} else if foundProjectRoot {
			break
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Cache miss for all visited directories to avoid redundant filesystem scans
	if r.cache != nil {
		for _, visitedDir := range visitedDirs {
			r.cache.Set(visitedDir, patterns, "")
		}
	}
	return "", false
}

func (r *Resolver) findConfigInDir(dir string, patterns []string) (string, bool) {
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			// filepath.Glob only returns ErrBadPattern for malformed patterns
			continue
		}
		if len(matches) > 0 {
			configPath := matches[0]
			if r.cache != nil {
				r.cache.Set(dir, patterns, configPath)
			}
			return configPath, true
		}
	}
	return "", false
}

func isProjectRoot(dir string) bool {
	return anyFileExists(dir, projectRootIndicators)
}

func anyFileExists(dir string, files []string) bool {
	for _, file := range files {
		if _, err := os.Stat(filepath.Join(dir, file)); err == nil {
			return true
		}
	}
	return false
}

// IsMonorepoRoot detects monorepo by checking for workspace indicators.
func IsMonorepoRoot(dir string) bool {
	if anyFileExists(dir, monorepoIndicators) {
		return true
	}

	packageJSON := filepath.Join(dir, "package.json")
	if content, err := os.ReadFile(packageJSON); err == nil {
		if workspacesPattern.Match(content) {
			return true
		}
	}
	return false
}
