package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDelete_RequiresConfirmation_NonInteractive(t *testing.T) {
	withTempConfig(t, seedDefaults())

	_, errOut, code := runAipim(t, "delete", "sgws")
	if code != ExitUsage {
		t.Fatalf("exit = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(errOut, "--yes") {
		t.Errorf("expected --yes hint in stderr, got: %s", errOut)
	}
}

func TestDelete_SuccessWithYes(t *testing.T) {
	withTempConfig(t, seedDefaults())

	out, _, code := runAipim(t, "delete", "sgws", "--yes")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(out, "Deleted") {
		t.Errorf("expected success message, got: %s", out)
	}

	// Verify the profile is actually gone via aipim list --quiet.
	out, _, code = runAipim(t, "list", "--quiet")
	if code != ExitOK {
		t.Fatalf("list exit = %d, want %d", code, ExitOK)
	}
	names := strings.Fields(out)
	for _, n := range names {
		if n == "sgws" {
			t.Errorf("profile sgws should have been deleted, still present: %v", names)
		}
	}
}

func TestDelete_ByAlias(t *testing.T) {
	withTempConfig(t, seedDefaults())

	_, _, code := runAipim(t, "delete", "s", "--yes")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
}

func TestDelete_Missing(t *testing.T) {
	withTempConfig(t, seedDefaults())

	_, errOut, code := runAipim(t, "delete", "nope", "--yes")
	if code != ExitUsage {
		t.Fatalf("exit = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(errOut, "does not exist") {
		t.Errorf("expected 'does not exist' in stderr, got: %s", errOut)
	}
}

func TestDelete_RemovesProfileDir(t *testing.T) {
	dir := withTempConfig(t, seedDefaults())

	// Create a profile with an explicit path inside the config dir, then delete it.
	profilePath := filepath.Join(dir, "profiles", "ephemeral")
	if err := os.MkdirAll(profilePath, 0o755); err != nil {
		t.Fatalf("seed profile dir: %v", err)
	}
	// Register the profile in config.
	_, _, code := runAipim(t,
		"create",
		"--no-tui",
		"--name", "ephemeral",
		"--agent", "Codex",
		"--path", profilePath,
	)
	if code != ExitOK {
		t.Fatalf("create exit = %d, want %d", code, ExitOK)
	}

	if _, err := os.Stat(profilePath); err != nil {
		t.Fatalf("profile dir should exist before delete: %v", err)
	}

	_, _, code = runAipim(t, "delete", "ephemeral", "--yes")
	if code != ExitOK {
		t.Fatalf("delete exit = %d, want %d", code, ExitOK)
	}
	if _, err := os.Stat(profilePath); !os.IsNotExist(err) {
		t.Errorf("profile dir should have been removed, got err: %v", err)
	}
}

func TestDelete_KeepFiles(t *testing.T) {
	dir := withTempConfig(t, seedDefaults())

	profilePath := filepath.Join(dir, "profiles", "kept")
	if err := os.MkdirAll(profilePath, 0o755); err != nil {
		t.Fatalf("seed profile dir: %v", err)
	}
	_, _, code := runAipim(t,
		"create",
		"--no-tui",
		"--name", "kept",
		"--agent", "Codex",
		"--path", profilePath,
	)
	if code != ExitOK {
		t.Fatalf("create exit = %d", code)
	}

	_, _, code = runAipim(t, "delete", "kept", "--yes", "--keep-files")
	if code != ExitOK {
		t.Fatalf("delete exit = %d", code)
	}
	if _, err := os.Stat(profilePath); err != nil {
		t.Errorf("profile dir should be preserved with --keep-files, got err: %v", err)
	}
}

func TestDelete_JSON(t *testing.T) {
	withTempConfig(t, seedDefaults())

	out, _, code := runAipim(t, "delete", "sgws", "--yes", "--json")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	var payload struct {
		Deleted string `json:"deleted"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, out)
	}
	if payload.Deleted != "sgws" {
		t.Errorf("deleted = %q, want sgws", payload.Deleted)
	}
}
