package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/victorseara/aipim/internal/agent"
	"github.com/victorseara/aipim/internal/profile"
)

// ErrConfigNotFound indicates that aipim has not been configured yet.
var ErrConfigNotFound = errors.New("config not found")

// ErrConfigCorrupt indicates that the persisted config file contains invalid JSON.
var ErrConfigCorrupt = errors.New("config is corrupt")

// AppConfig is the persisted aipim application configuration.
type AppConfig struct {
	DefaultAgentName string            `json:"default_agent"`
	Profiles         []profile.Profile `json:"profiles"`
	Agents           []agent.Agent     `json:"agents"`
}

// DefaultConfig returns a config initialized with the built-in agents registry.
func DefaultConfig() *AppConfig {
	return &AppConfig{
		Agents: agent.BuiltIns(),
	}
}

// Load reads the aipim config from disk.
func Load() (*AppConfig, error) {
	configPath, err := ConfigFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrConfigNotFound
		}

		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfigCorrupt, err)
	}

	cfg.Agents = agent.EnsureBuiltIns(cfg.Agents)
	if cfg.Profiles == nil {
		cfg.Profiles = []profile.Profile{}
	}

	return &cfg, nil
}

// Save writes the aipim config to disk.
func (c *AppConfig) Save() error {
	configDir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	configPath, err := ConfigFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// BackupCorruptConfig moves the current config aside before rebuilding it.
func BackupCorruptConfig() (string, error) {
	configPath, err := ConfigFilePath()
	if err != nil {
		return "", err
	}

	backupPath := filepath.Join(filepath.Dir(configPath), fmt.Sprintf("config.corrupt-%s.json", time.Now().Format("20060102-150405")))
	if err := os.Rename(configPath, backupPath); err != nil {
		return "", fmt.Errorf("backup corrupt config: %w", err)
	}

	return backupPath, nil
}
