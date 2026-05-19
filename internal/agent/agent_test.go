package agent

import (
	"strings"
	"testing"
)

func TestBuiltInsAreClonedAndComplete(t *testing.T) {
	expected := map[string]string{
		"Claude Code": "claude",
		"Codex":       "codex",
		"Gemini":      "gemini",
		"OpenCode":    "opencode",
		"Copilot":     "gh copilot",
	}

	got := BuiltIns()
	if len(got) != len(expected) {
		t.Fatalf("expected %d built-ins, got %d", len(expected), len(got))
	}
	for _, ag := range got {
		if !ag.IsBuiltIn {
			t.Errorf("agent %q should be marked built-in", ag.Name)
		}
		wantCmd, ok := expected[ag.Name]
		if !ok {
			t.Errorf("unexpected agent %q", ag.Name)
			continue
		}
		if ag.LaunchCmd != wantCmd {
			t.Errorf("agent %q: launch cmd %q, want %q", ag.Name, ag.LaunchCmd, wantCmd)
		}
	}

	got[0].Name = "MUTATED"
	if BuiltIns()[0].Name == "MUTATED" {
		t.Fatal("BuiltIns returned a shared slice; mutation leaked")
	}
}

func TestConfigEnvVar(t *testing.T) {
	cases := map[string]string{
		"claude":       "CLAUDE_CONFIG_DIR",
		"  claude  ":   "CLAUDE_CONFIG_DIR",
		"CLAUDE":       "CLAUDE_CONFIG_DIR",
		"codex":        "XDG_CONFIG_HOME",
		"gemini":       "XDG_CONFIG_HOME",
		"opencode":     "XDG_CONFIG_HOME",
		"gh copilot":   "XDG_CONFIG_HOME",
		"GH   COPILOT": "XDG_CONFIG_HOME",
		"unknown-bin":  "XDG_CONFIG_HOME",
	}

	for input, want := range cases {
		if got := ConfigEnvVar(input); got != want {
			t.Errorf("ConfigEnvVar(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestUpsertRejectsBuiltInName(t *testing.T) {
	agents := BuiltIns()
	if _, err := Upsert(agents, Agent{Name: "Claude Code", LaunchCmd: "fake"}); err == nil {
		t.Fatal("expected error modifying a built-in agent")
	}
}

func TestUpsertAddsCustomAgent(t *testing.T) {
	agents := BuiltIns()
	updated, err := Upsert(agents, Agent{Name: "Custom", LaunchCmd: "custom-cmd"})
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	if len(updated) != len(agents)+1 {
		t.Fatalf("expected %d agents, got %d", len(agents)+1, len(updated))
	}
	added, ok := FindByName(updated, "Custom")
	if !ok {
		t.Fatal("custom agent not found after upsert")
	}
	if added.IsBuiltIn {
		t.Fatal("custom agent should not be marked built-in")
	}
	if added.LaunchCmd != "custom-cmd" {
		t.Fatalf("LaunchCmd %q, want custom-cmd", added.LaunchCmd)
	}
}

func TestUpsertUpdatesExistingCustom(t *testing.T) {
	agents := []Agent{{Name: "Custom", LaunchCmd: "old"}}
	updated, err := Upsert(agents, Agent{Name: "Custom", LaunchCmd: "new"})
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	if len(updated) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(updated))
	}
	if updated[0].LaunchCmd != "new" {
		t.Fatalf("LaunchCmd %q, want new", updated[0].LaunchCmd)
	}
}

func TestUpsertValidates(t *testing.T) {
	if _, err := Upsert(nil, Agent{Name: "", LaunchCmd: "x"}); err == nil {
		t.Fatal("expected error on empty name")
	}
	if _, err := Upsert(nil, Agent{Name: "x", LaunchCmd: ""}); err == nil {
		t.Fatal("expected error on empty launch command")
	}
}

func TestDeleteCustomByName(t *testing.T) {
	agents := []Agent{
		{Name: "Claude Code", LaunchCmd: "claude", IsBuiltIn: true},
		{Name: "Custom", LaunchCmd: "custom"},
	}

	if _, err := DeleteCustomByName(agents, "Claude Code"); err == nil {
		t.Fatal("expected error deleting built-in")
	}
	if _, err := DeleteCustomByName(agents, "missing"); err == nil {
		t.Fatal("expected error deleting nonexistent agent")
	}
	updated, err := DeleteCustomByName(agents, "Custom")
	if err != nil {
		t.Fatalf("DeleteCustomByName failed: %v", err)
	}
	if len(updated) != 1 || updated[0].Name != "Claude Code" {
		t.Fatalf("unexpected agents after delete: %+v", updated)
	}
}

func TestEnsureBuiltIns(t *testing.T) {
	// start with one built-in missing
	partial := []Agent{
		{Name: "Claude Code", LaunchCmd: "claude", IsBuiltIn: true},
		{Name: "Custom", LaunchCmd: "x"},
	}
	merged := EnsureBuiltIns(partial)

	expectedNames := []string{"Claude Code", "Codex", "Gemini", "OpenCode", "Copilot", "Custom"}
	for _, want := range expectedNames {
		if _, ok := FindByName(merged, want); !ok {
			t.Errorf("EnsureBuiltIns missing %q", want)
		}
	}

	// running again should be idempotent
	if len(EnsureBuiltIns(merged)) != len(merged) {
		t.Fatal("EnsureBuiltIns is not idempotent")
	}
}

func TestFindByNameCaseAndWhitespace(t *testing.T) {
	agents := []Agent{{Name: "Claude Code"}}
	if _, ok := FindByName(agents, "  claude code  "); !ok {
		t.Fatal("FindByName should normalize whitespace and case")
	}
	if _, ok := FindByName(agents, "claude"); ok {
		t.Fatal("FindByName should not partial-match")
	}
}

func TestConfigEnvVarPureFn(t *testing.T) {
	// regression: make sure ConfigEnvVar does not panic on empty input
	if got := ConfigEnvVar(""); !strings.HasSuffix(got, "_CONFIG_HOME") {
		t.Fatalf("ConfigEnvVar(\"\") = %q, want fallback", got)
	}
}
