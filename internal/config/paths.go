package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	// EnvConfigHome is the environment variable that overrides the aipim config directory.
	// Takes precedence over XDG_CONFIG_HOME and the default ~/.config/aipim path.
	EnvConfigHome = "AIPIM_CONFIG_HOME"

	// EnvXDGConfigHome is the standard XDG base directory env var.
	EnvXDGConfigHome = "XDG_CONFIG_HOME"
)

var (
	overrideMu  sync.RWMutex
	overrideDir string
)

// SetConfigDirOverride forces ConfigDir/ConfigFilePath/ProfilesDir to resolve to the given path.
// Pass an empty string to clear the override. Intended for the --config-dir CLI flag and tests.
func SetConfigDirOverride(dir string) {
	overrideMu.Lock()
	defer overrideMu.Unlock()
	overrideDir = strings.TrimSpace(dir)
}

func currentOverride() string {
	overrideMu.RLock()
	defer overrideMu.RUnlock()
	return overrideDir
}

// ConfigDir returns the aipim configuration directory.
// Resolution order: SetConfigDirOverride > $AIPIM_CONFIG_HOME > $XDG_CONFIG_HOME/aipim > ~/.config/aipim
func ConfigDir() (string, error) {
	if override := currentOverride(); override != "" {
		return override, nil
	}

	if env := strings.TrimSpace(os.Getenv(EnvConfigHome)); env != "" {
		return env, nil
	}

	if xdg := strings.TrimSpace(os.Getenv(EnvXDGConfigHome)); xdg != "" {
		return filepath.Join(xdg, "aipim"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	return filepath.Join(home, ".config", "aipim"), nil
}

// ConfigFilePath returns the aipim config.json path.
func ConfigFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "config.json"), nil
}

// ProfilesDir returns the default profiles directory.
func ProfilesDir() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "profiles"), nil
}

// DefaultProfilePath returns the default path for a named profile.
func DefaultProfilePath(name string) (string, error) {
	profilesDir, err := ProfilesDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(profilesDir, strings.TrimSpace(name)), nil
}

// ExpandPath expands a leading ~/ prefix into the user's home directory.
func ExpandPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", nil
	}

	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}

		if trimmed == "~" {
			return home, nil
		}

		return filepath.Join(home, strings.TrimPrefix(trimmed, "~/")), nil
	}

	return trimmed, nil
}
