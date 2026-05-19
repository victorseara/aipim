package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/victorseara/aipim/internal/agent"
	"github.com/victorseara/aipim/internal/profile"
)

func withHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	return dir
}

func TestExpandPath(t *testing.T) {
	home := withHome(t)

	cases := map[string]string{
		"":             "",
		"  ":           "",
		"~":            home,
		"~/foo/bar":    filepath.Join(home, "foo/bar"),
		"/abs/path":    "/abs/path",
		"relative/dir": "relative/dir",
	}

	for input, want := range cases {
		got, err := ExpandPath(input)
		if err != nil {
			t.Fatalf("ExpandPath(%q) error: %v", input, err)
		}
		if got != want {
			t.Errorf("ExpandPath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestConfigDirAndFilePath(t *testing.T) {
	home := withHome(t)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	if want := filepath.Join(home, ".config", "aipim"); dir != want {
		t.Fatalf("ConfigDir = %q, want %q", dir, want)
	}

	file, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath: %v", err)
	}
	if !strings.HasSuffix(file, filepath.Join(".config", "aipim", "config.json")) {
		t.Fatalf("ConfigFilePath = %q", file)
	}

	profiles, err := ProfilesDir()
	if err != nil {
		t.Fatalf("ProfilesDir: %v", err)
	}
	if !strings.HasSuffix(profiles, filepath.Join(".config", "aipim", "profiles")) {
		t.Fatalf("ProfilesDir = %q", profiles)
	}

	defaultProfile, err := DefaultProfilePath("work")
	if err != nil {
		t.Fatalf("DefaultProfilePath: %v", err)
	}
	if !strings.HasSuffix(defaultProfile, filepath.Join("profiles", "work")) {
		t.Fatalf("DefaultProfilePath = %q", defaultProfile)
	}
}

func TestLoadReturnsErrConfigNotFound(t *testing.T) {
	withHome(t)
	if _, err := Load(); !errors.Is(err, ErrConfigNotFound) {
		t.Fatalf("expected ErrConfigNotFound, got %v", err)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	withHome(t)

	cfg := DefaultConfig()
	cfg.DefaultAgentName = "Claude Code"
	cfg.Profiles = []profile.Profile{{
		Name:      "work",
		Path:      "/tmp/work",
		AgentName: "Claude Code",
		CreatedAt: "2026-01-01T00:00:00Z",
	}}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.DefaultAgentName != "Claude Code" {
		t.Errorf("DefaultAgentName = %q", loaded.DefaultAgentName)
	}
	if len(loaded.Profiles) != 1 || loaded.Profiles[0].Name != "work" {
		t.Errorf("profiles = %+v", loaded.Profiles)
	}
	// EnsureBuiltIns should keep all built-ins
	if _, ok := agent.FindByName(loaded.Agents, "Codex"); !ok {
		t.Error("Codex built-in missing after load")
	}
}

func TestLoadCorruptConfig(t *testing.T) {
	withHome(t)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath: %v", err)
	}
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write corrupt config: %v", err)
	}

	if _, err := Load(); !errors.Is(err, ErrConfigCorrupt) {
		t.Fatalf("expected ErrConfigCorrupt, got %v", err)
	}
}

func TestBackupCorruptConfig(t *testing.T) {
	withHome(t)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath: %v", err)
	}
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	backup, err := BackupCorruptConfig()
	if err != nil {
		t.Fatalf("BackupCorruptConfig: %v", err)
	}
	if _, err := os.Stat(backup); err != nil {
		t.Fatalf("backup file missing: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("original config should have been moved; stat err = %v", err)
	}
	if !strings.Contains(backup, "config.corrupt-") {
		t.Errorf("unexpected backup name: %s", backup)
	}
}

func TestSaveWritesValidJSON(t *testing.T) {
	withHome(t)
	cfg := DefaultConfig()
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	path, _ := ConfigFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.HasPrefix(string(data), "{") || !strings.HasSuffix(string(data), "}\n") {
		t.Fatalf("config.json should be pretty-printed JSON ending in newline; got: %q", string(data))
	}
	if !strings.Contains(string(data), `"default_agent"`) {
		t.Error("expected default_agent key in saved config")
	}
}
