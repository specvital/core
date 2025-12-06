//go:build integration

package integration

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Repository represents a target repository for integration testing.
type Repository struct {
	Framework string `yaml:"framework"`
	Name      string `yaml:"name"`
	Ref       string `yaml:"ref"`
	URL       string `yaml:"url"`
}

// ReposConfig holds the list of repositories to test.
type ReposConfig struct {
	Repositories []Repository `yaml:"repositories"`
}

// LoadRepos loads repository definitions from repos.yaml.
func LoadRepos() (*ReposConfig, error) {
	testDataDir, err := getTestDataDir()
	if err != nil {
		return nil, err
	}
	reposPath := filepath.Join(testDataDir, "..", "repos.yaml")
	return loadReposFromPath(reposPath)
}

func loadReposFromPath(path string) (*ReposConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read repos config from %s: %w", path, err)
	}

	var config ReposConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("unmarshal repos config: %w", err)
	}

	if err := validateReposConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid repos config: %w", err)
	}

	return &config, nil
}

func validateReposConfig(config *ReposConfig) error {
	if len(config.Repositories) == 0 {
		return errors.New("no repositories defined")
	}

	for i, repo := range config.Repositories {
		if repo.Name == "" {
			return fmt.Errorf("repository %d: name is required", i)
		}
		if repo.URL == "" {
			return fmt.Errorf("repository %s: url is required", repo.Name)
		}
		if repo.Ref == "" {
			return fmt.Errorf("repository %s: ref is required", repo.Name)
		}
		if repo.Framework == "" {
			return fmt.Errorf("repository %s: framework is required", repo.Name)
		}
	}
	return nil
}

func getTestDataDir() (string, error) {
	integrationDir, err := getIntegrationDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(integrationDir, "testdata"), nil
}

func getIntegrationDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	return wd, nil
}
