package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/victorseara/aipim/internal/agent"
)

func TestAgent_List_Table(t *testing.T) {
	withTempConfig(t, seedDefaults())

	out, _, code := runAipim(t, "agent", "list")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	for _, want := range []string{"Claude Code", "Codex", "Gemini", "OpenCode", "Copilot"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing built-in %q in output:\n%s", want, out)
		}
	}
	// Default agent should be marked.
	if !strings.Contains(out, "*") {
		t.Errorf("expected default-agent marker '*' in output, got:\n%s", out)
	}
}

func TestAgent_List_JSON(t *testing.T) {
	withTempConfig(t, seedDefaults())

	out, _, code := runAipim(t, "agent", "list", "--json")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	var payload struct {
		DefaultAgent string        `json:"default_agent"`
		Agents       []agent.Agent `json:"agents"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, out)
	}
	if payload.DefaultAgent != "Claude Code" {
		t.Errorf("default_agent = %q, want Claude Code", payload.DefaultAgent)
	}
	if len(payload.Agents) < 5 {
		t.Errorf("expected at least 5 built-in agents, got %d", len(payload.Agents))
	}
}

func TestAgent_Add_New(t *testing.T) {
	withTempConfig(t, seedDefaults())

	_, _, code := runAipim(t,
		"agent", "add",
		"--name", "My Agent",
		"--cmd", "/usr/local/bin/my-agent --fast",
	)
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}

	// Verify via aipim agent list --json that the custom agent is there.
	out, _, code := runAipim(t, "agent", "list", "--json")
	if code != ExitOK {
		t.Fatalf("list exit = %d", code)
	}
	if !strings.Contains(out, `"name": "My Agent"`) {
		t.Errorf("custom agent missing from list:\n%s", out)
	}
	if !strings.Contains(out, `"built_in": false`) {
		t.Errorf("custom agent should be flagged built_in=false; got:\n%s", out)
	}
}

func TestAgent_Add_MissingFlags(t *testing.T) {
	withTempConfig(t, seedDefaults())

	_, errOut, code := runAipim(t, "agent", "add", "--name", "X")
	if code != ExitUsage {
		t.Fatalf("exit = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(errOut, "--cmd is required") {
		t.Errorf("expected --cmd required hint, got: %s", errOut)
	}
}

func TestAgent_Rm_BuiltIn_Rejected(t *testing.T) {
	withTempConfig(t, seedDefaults())

	_, errOut, code := runAipim(t, "agent", "rm", "Claude Code")
	// Default agent guard fires first (Claude Code is the default).
	if code != ExitUsage {
		t.Fatalf("exit = %d, want %d (default-agent guard)", code, ExitUsage)
	}
	if !strings.Contains(errOut, "default agent") {
		t.Errorf("expected default-agent error, got: %s", errOut)
	}
}

func TestAgent_Rm_BuiltIn_NotDefault(t *testing.T) {
	// Codex isn't the default; the built-in guard should fire.
	cfg := seedDefaults()
	// Detach Codex from any profile so the "in use" check doesn't fire.
	cfg.Profiles = cfg.Profiles[:1] // keep only sgws
	withTempConfig(t, cfg)

	_, errOut, code := runAipim(t, "agent", "rm", "Codex")
	if code != ExitConfig {
		t.Fatalf("exit = %d, want %d", code, ExitConfig)
	}
	if !strings.Contains(errOut, "built-in") {
		t.Errorf("expected built-in guard message, got: %s", errOut)
	}
}

func TestAgent_Rm_Custom(t *testing.T) {
	cfg := seedDefaults()
	cfg.Agents = append(cfg.Agents, agent.Agent{Name: "My Agent", LaunchCmd: "/usr/local/bin/my-agent"})
	withTempConfig(t, cfg)

	_, _, code := runAipim(t, "agent", "rm", "My Agent")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}

	out, _, _ := runAipim(t, "agent", "list", "--quiet")
	if strings.Contains(out, "My Agent") {
		t.Errorf("custom agent should have been removed:\n%s", out)
	}
}

func TestAgent_Rm_InUse(t *testing.T) {
	cfg := seedDefaults()
	cfg.Agents = append(cfg.Agents, agent.Agent{Name: "My Agent", LaunchCmd: "/usr/local/bin/my-agent"})
	cfg.Profiles = append(cfg.Profiles, newProfile("using", "", "My Agent", "depends on My Agent"))
	withTempConfig(t, cfg)

	_, errOut, code := runAipim(t, "agent", "rm", "My Agent")
	if code != ExitUsage {
		t.Fatalf("exit = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(errOut, "still used by") {
		t.Errorf("expected in-use error, got: %s", errOut)
	}
}

func TestAgent_Default_Success(t *testing.T) {
	withTempConfig(t, seedDefaults())

	_, _, code := runAipim(t, "agent", "default", "Codex")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}

	out, _, _ := runAipim(t, "agent", "list", "--json")
	var payload struct {
		DefaultAgent string `json:"default_agent"`
	}
	_ = json.Unmarshal([]byte(out), &payload)
	if payload.DefaultAgent != "Codex" {
		t.Errorf("default_agent = %q, want Codex", payload.DefaultAgent)
	}
}

func TestAgent_Default_Unknown(t *testing.T) {
	withTempConfig(t, seedDefaults())

	_, errOut, code := runAipim(t, "agent", "default", "Bogus")
	if code != ExitConfig {
		t.Fatalf("exit = %d, want %d", code, ExitConfig)
	}
	if !strings.Contains(errOut, "not registered") {
		t.Errorf("expected not-registered error, got: %s", errOut)
	}
}
