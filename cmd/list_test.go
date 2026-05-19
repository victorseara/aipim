package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/victorseara/aipim/internal/config"
	"github.com/victorseara/aipim/internal/profile"
)

func TestList_EmptyConfig(t *testing.T) {
	withTempConfig(t, seedEmpty())

	out, _, code := runAipim(t, "list")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(out, "No profiles configured") {
		t.Errorf("expected empty-state message in stdout, got %q", out)
	}
}

func TestList_Table(t *testing.T) {
	withTempConfig(t, seedDefaults())

	out, _, code := runAipim(t, "list")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	for _, want := range []string{"NAME", "sgws", "personal", "Claude Code", "Codex"} {
		if !strings.Contains(out, want) {
			t.Errorf("table output missing %q.\nGot:\n%s", want, out)
		}
	}
}

func TestList_JSON(t *testing.T) {
	withTempConfig(t, seedDefaults())

	out, _, code := runAipim(t, "list", "--json")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}

	var payload struct {
		DefaultAgent string            `json:"default_agent"`
		Profiles     []profile.Profile `json:"profiles"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, out)
	}
	if payload.DefaultAgent != "Claude Code" {
		t.Errorf("default_agent = %q, want Claude Code", payload.DefaultAgent)
	}
	if len(payload.Profiles) != 2 {
		t.Fatalf("len(profiles) = %d, want 2", len(payload.Profiles))
	}
	if payload.Profiles[0].Name != "sgws" {
		t.Errorf("profiles[0].name = %q, want sgws", payload.Profiles[0].Name)
	}
	if !strings.Contains(payload.Profiles[0].Description, "Snowflake") {
		t.Errorf("profiles[0].description missing keyword; got %q", payload.Profiles[0].Description)
	}
}

func TestList_Quiet(t *testing.T) {
	withTempConfig(t, seedDefaults())

	out, _, code := runAipim(t, "list", "--quiet")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	got := strings.Split(strings.TrimSpace(out), "\n")
	want := []string{"sgws", "personal"}
	if len(got) != len(want) {
		t.Fatalf("quiet output lines = %d, want %d\nOutput: %q", len(got), len(want), out)
	}
	for i, name := range want {
		if got[i] != name {
			t.Errorf("line %d = %q, want %q", i, got[i], name)
		}
	}
}

func TestList_WithDescription(t *testing.T) {
	withTempConfig(t, seedDefaults())

	out, _, code := runAipim(t, "list", "--with-description")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(out, "DESCRIPTION") {
		t.Errorf("expected DESCRIPTION column header, got:\n%s", out)
	}
	if !strings.Contains(out, "Snowflake MCP") {
		t.Errorf("expected truncated description in output, got:\n%s", out)
	}
}

func TestList_NoConfig(t *testing.T) {
	// No seed: AIPIM_CONFIG_HOME exists but is empty.
	withTempConfig(t, nil)

	out, _, code := runAipim(t, "list", "--json")
	if code != ExitConfig {
		t.Fatalf("exit = %d, want %d (config error)\nOutput: %s", code, ExitConfig, out)
	}
	// JSON envelope on stdout when --json is set.
	if !strings.Contains(out, `"code":3`) {
		t.Errorf("expected JSON error envelope with code 3, got: %s", out)
	}
}

// seedEmpty returns an empty config that still has the built-in agents registered.
func seedEmpty() *config.AppConfig {
	cfg := *seedDefaults()
	cfg.Profiles = nil
	return &cfg
}
