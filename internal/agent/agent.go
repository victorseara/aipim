package agent

import (
	"errors"
	"fmt"
	"strings"
)

// Agent describes an AI coding agent that can be launched by aipim.
type Agent struct {
	Name      string `json:"name"`
	LaunchCmd string `json:"launch_cmd"`
	IsBuiltIn bool   `json:"built_in"`
}

var builtInAgents = []Agent{
	{Name: "Claude Code", LaunchCmd: "claude", IsBuiltIn: true},
	{Name: "Codex", LaunchCmd: "codex", IsBuiltIn: true},
	{Name: "Gemini", LaunchCmd: "gemini", IsBuiltIn: true},
	{Name: "OpenCode", LaunchCmd: "opencode", IsBuiltIn: true},
	{Name: "Copilot", LaunchCmd: "gh copilot", IsBuiltIn: true},
}

var configEnvVars = map[string]string{
	"claude":     "CLAUDE_CONFIG_DIR",
	"codex":      "XDG_CONFIG_HOME",
	"gemini":     "XDG_CONFIG_HOME",
	"opencode":   "XDG_CONFIG_HOME",
	"gh copilot": "XDG_CONFIG_HOME",
}

// BuiltIns returns the built-in agent registry.
func BuiltIns() []Agent {
	cloned := make([]Agent, len(builtInAgents))
	copy(cloned, builtInAgents)
	return cloned
}

// ConfigEnvVar returns the configuration environment variable for a launch command.
func ConfigEnvVar(launchCmd string) string {
	command := strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(launchCmd))), " ")
	if envVar, ok := configEnvVars[command]; ok {
		return envVar
	}

	return "XDG_CONFIG_HOME"
}

// FindByName looks up an agent by display name.
func FindByName(agents []Agent, name string) (Agent, bool) {
	for _, candidate := range agents {
		if strings.EqualFold(strings.TrimSpace(candidate.Name), strings.TrimSpace(name)) {
			return candidate, true
		}
	}

	return Agent{}, false
}

// Upsert registers a new custom agent or updates an existing custom agent with the same name.
func Upsert(agents []Agent, candidate Agent) ([]Agent, error) {
	candidate.Name = strings.TrimSpace(candidate.Name)
	candidate.LaunchCmd = strings.TrimSpace(candidate.LaunchCmd)

	if candidate.Name == "" {
		return nil, errors.New("agent name cannot be empty")
	}
	if candidate.LaunchCmd == "" {
		return nil, errors.New("agent launch command cannot be empty")
	}

	for i, existing := range agents {
		if !strings.EqualFold(existing.Name, candidate.Name) {
			continue
		}
		if existing.IsBuiltIn {
			return nil, fmt.Errorf("built-in agent %q cannot be modified", existing.Name)
		}

		candidate.IsBuiltIn = false
		updated := make([]Agent, len(agents))
		copy(updated, agents)
		updated[i] = candidate
		return updated, nil
	}

	candidate.IsBuiltIn = false
	return append(append([]Agent(nil), agents...), candidate), nil
}

// DeleteCustomByName removes a custom agent by name.
func DeleteCustomByName(agents []Agent, name string) ([]Agent, error) {
	for i, existing := range agents {
		if !strings.EqualFold(existing.Name, strings.TrimSpace(name)) {
			continue
		}
		if existing.IsBuiltIn {
			return nil, fmt.Errorf("built-in agent %q cannot be deleted", existing.Name)
		}

		updated := append([]Agent(nil), agents[:i]...)
		updated = append(updated, agents[i+1:]...)
		return updated, nil
	}

	return nil, fmt.Errorf("agent %q does not exist", name)
}

// EnsureBuiltIns makes sure all built-in agents are present in the registry.
func EnsureBuiltIns(agents []Agent) []Agent {
	merged := append([]Agent(nil), agents...)

	for _, builtIn := range builtInAgents {
		if _, ok := FindByName(merged, builtIn.Name); ok {
			continue
		}
		merged = append(merged, builtIn)
	}

	return merged
}
