package framework

import (
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// ConfigScope defines the effective scope of a config file.
// Handles root resolution, include/exclude patterns, and workspace projects.
type ConfigScope struct {
	ConfigPath      string
	BaseDir         string
	Include         []string
	Exclude         []string
	Settings        map[string]interface{}
	Projects        []ProjectScope
	Framework       string
	TestPatterns    []string
	ExcludePatterns []string
	RootDir         string

	// Roots contains additional root directories (e.g., Jest's roots config).
	// When set, Contains() checks if a file is within any of these roots.
	Roots []string

	// GlobalsMode: when true, test files don't need explicit imports (e.g., Jest default).
	GlobalsMode bool
}

type ProjectScope struct {
	Name     string
	BaseDir  string
	Include  []string
	Exclude  []string
	Settings map[string]interface{}
}

// AggregatedProjectScope aggregates multiple ConfigScope instances for hierarchical config resolution.
type AggregatedProjectScope struct {
	Configs     map[string]*ConfigScope
	ConfigFiles []string
}

func NewProjectScope() *AggregatedProjectScope {
	return &AggregatedProjectScope{
		Configs:     make(map[string]*ConfigScope),
		ConfigFiles: []string{},
	}
}

func (ps *AggregatedProjectScope) AddConfig(path string, scope *ConfigScope) {
	ps.Configs[path] = scope
	ps.ConfigFiles = append(ps.ConfigFiles, path)
}

func (ps *AggregatedProjectScope) FindConfig(path string) *ConfigScope {
	return ps.Configs[path]
}

func (ps *AggregatedProjectScope) HasConfigFile(filename string) bool {
	for _, path := range ps.ConfigFiles {
		if path == filename {
			return true
		}
	}
	return false
}

// Contains checks if filePath is within this config's scope.
func (s *ConfigScope) Contains(filePath string) bool {
	if s == nil {
		return false
	}

	filePath = filepath.Clean(filePath)
	filePath = filepath.ToSlash(filePath)

	roots := s.effectiveRoots()
	for _, r := range roots {
		root := filepath.ToSlash(filepath.Clean(r))

		relPath, err := filepath.Rel(root, filePath)
		if err != nil {
			// Skip this root if relative path calculation fails
			// (e.g., different drive letters on Windows)
			continue
		}
		relPath = filepath.ToSlash(relPath)

		if strings.HasPrefix(relPath, "..") {
			continue
		}

		if len(s.Include) > 0 {
			matched := false
			for _, pattern := range s.Include {
				if match, err := doublestar.Match(pattern, relPath); err == nil && match {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		excluded := false
		for _, pattern := range s.Exclude {
			if match, err := doublestar.Match(pattern, relPath); err == nil && match {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		return true
	}

	return false
}

func (s *ConfigScope) effectiveRoots() []string {
	if len(s.Roots) > 0 {
		return s.Roots
	}

	roots := []string{s.BaseDir}
	for _, p := range s.Projects {
		if p.BaseDir != "" {
			roots = append(roots, p.BaseDir)
		}
	}
	return roots
}

// Depth returns the directory depth of BaseDir (used for selecting nearest config).
func (s *ConfigScope) Depth() int {
	if s == nil || s.BaseDir == "" {
		return 0
	}

	baseDir := filepath.ToSlash(filepath.Clean(s.BaseDir))
	if baseDir == "." || baseDir == "/" {
		return 0
	}

	return strings.Count(baseDir, "/")
}

// FindMatchingProject finds the most specific project for a file.
func (s *ConfigScope) FindMatchingProject(filePath string) *ProjectScope {
	if s == nil || len(s.Projects) == 0 {
		return nil
	}

	filePath = filepath.ToSlash(filepath.Clean(filePath))

	var bestMatch *ProjectScope
	bestDepth := -1

	for i := range s.Projects {
		project := &s.Projects[i]
		baseDir := filepath.ToSlash(filepath.Clean(project.BaseDir))

		relPath, err := filepath.Rel(baseDir, filePath)
		if err != nil {
			continue
		}
		relPath = filepath.ToSlash(relPath)

		if strings.HasPrefix(relPath, "..") {
			continue
		}

		if len(project.Include) > 0 {
			matched := false
			for _, pattern := range project.Include {
				if match, err := doublestar.Match(pattern, relPath); err == nil && match {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		excluded := false
		for _, pattern := range project.Exclude {
			if match, err := doublestar.Match(pattern, relPath); err == nil && match {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		depth := strings.Count(baseDir, "/")
		if depth > bestDepth {
			bestDepth = depth
			bestMatch = project
		}
	}

	return bestMatch
}

// NewConfigScope creates a ConfigScope with root resolved relative to config directory.
func NewConfigScope(configPath string, root string) *ConfigScope {
	configDir := filepath.Dir(configPath)

	var baseDir string
	if root != "" {
		baseDir = filepath.Clean(filepath.Join(configDir, root))
	} else {
		baseDir = configDir
	}

	return &ConfigScope{
		ConfigPath: configPath,
		BaseDir:    baseDir,
		Settings:   make(map[string]interface{}),
	}
}
