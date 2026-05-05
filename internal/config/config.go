package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the .keysync.json configuration file.
type Config struct {
	Repos map[string]RepoConfig `json:"repos"`
}

// RepoConfig maps a GitHub repo to its project name, global keys, and platforms.
type RepoConfig struct {
	Project   string         `json:"project"`
	Globals   []string       `json:"globals,omitempty"`
	Platforms PlatformConfig `json:"platforms"`
}

// PlatformConfig holds optional configuration for each deployment platform.
type PlatformConfig struct {
	Vercel   *VercelConfig   `json:"vercel,omitempty"`
	Railway  *RailwayConfig  `json:"railway,omitempty"`
	Supabase *SupabaseConfig `json:"supabase,omitempty"`
}

// VercelConfig describes a Vercel project target.
type VercelConfig struct {
	ProjectID string   `json:"projectId"`
	Target    []string `json:"target,omitempty"` // "production", "preview", "development"
}

// RailwayConfig describes a Railway project target.
type RailwayConfig struct {
	Environment string `json:"environment,omitempty"`
	Service     string `json:"service,omitempty"`
}

// SupabaseConfig describes a Supabase project target.
type SupabaseConfig struct {
	Ref string `json:"ref"`
}

// DefaultConfig returns a minimal default configuration scaffold.
func DefaultConfig() *Config {
	return &Config{
		Repos: make(map[string]RepoConfig),
	}
}

// LoadConfig reads and parses .keysync.json from the given directory.
// It searches for the file in dir and its parent directories.
func LoadConfig(dir string) (*Config, string, error) {
	path, err := findConfig(dir)
	if err != nil {
		return nil, "", err
	}
	if path == "" {
		return DefaultConfig(), "", nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read config: %w", err)
	}
	cfg := DefaultConfig()
	if err := json.Unmarshal(raw, cfg); err != nil {
		return nil, "", fmt.Errorf("parse config: %w", err)
	}
	return cfg, path, nil
}

// SaveConfig writes the config to the specified path.
func SaveConfig(cfg *Config, path string) error {
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, raw, 0600)
}

// FindRepoByProject finds the repo key that contains the given project name.
// Returns the repo key, the config, and whether it was found.
func FindRepoByProject(cfg *Config, project string) (string, *RepoConfig, bool) {
	for repoKey, rc := range cfg.Repos {
		if rc.Project == project {
			return repoKey, &rc, true
		}
	}
	return "", nil, false
}

// findConfig searches for .keysync.json in dir and parent directories.
func findConfig(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(abs, ".keysync.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			break // reached root
		}
		abs = parent
	}
	return "", nil // not found
}

// DefaultConfigPath returns the default path for a new .keysync.json.
func DefaultConfigPath(dir string) string {
	return filepath.Join(dir, ".keysync.json")
}
