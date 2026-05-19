package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/victorseara/aipim/internal/agent"
	"github.com/victorseara/aipim/internal/config"
	"github.com/victorseara/aipim/internal/profile"
)

// RunEditFlow runs the interactive profile edit flow for the profile identified
// by originalName. Name/alias uniqueness is validated against the other
// profiles so the user can keep any field unchanged.
func RunEditFlow(cfg *config.AppConfig, originalName string) (*profile.Profile, error) {
	original, index, ok := profile.FindByName(cfg.Profiles, originalName)
	if !ok {
		return nil, fmt.Errorf("profile %q not found", originalName)
	}

	others := profilesExcluding(cfg.Profiles, index)

	name := original.Name
	nameForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Profile name").
				Description("Rename the profile or keep it unchanged.").
				Value(&name).
				Validate(func(value string) error {
					return profile.ValidateName(value, others)
				}),
		),
	)
	if err := nameForm.Run(); err != nil {
		return nil, err
	}
	trimmedName := strings.TrimSpace(name)

	alias := original.Alias
	aliasForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Profile alias").
				Description("Optional shortcut for `aipim <alias>`. Leave empty to clear.").
				Value(&alias).
				Validate(func(value string) error {
					return profile.ValidateAlias(value, trimmedName, others)
				}),
		),
	)
	if err := aliasForm.Run(); err != nil {
		return nil, err
	}
	trimmedAlias := strings.TrimSpace(alias)

	path := original.Path
	pathForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Config directory").
				Description("Leave empty to let the agent manage its own config directory.").
				Value(&path).
				Validate(func(value string) error {
					_, err := config.ExpandPath(value)
					return err
				}),
		),
	)
	if err := pathForm.Run(); err != nil {
		return nil, err
	}

	expandedPath, err := config.ExpandPath(path)
	if err != nil {
		return nil, err
	}
	var absolutePath string
	if strings.TrimSpace(expandedPath) != "" {
		absolutePath, err = filepath.Abs(expandedPath)
		if err != nil {
			return nil, fmt.Errorf("resolve absolute path %q: %w", expandedPath, err)
		}
	}

	description := original.Description
	descriptionForm := huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("Description").
				Description("Optional. Helps AI agents pick this profile from a user's prompt.").
				Lines(4).
				Value(&description),
		),
	)
	if err := descriptionForm.Run(); err != nil {
		return nil, err
	}
	trimmedDescription := strings.TrimSpace(description)

	selectedAgentName, err := selectAgentName(
		"Agent",
		"Change the agent or keep the current one.",
		cfg.Agents,
		true,
		original.AgentName,
	)
	if err != nil {
		return nil, err
	}

	agentName := selectedAgentName
	if selectedAgentName == customAgentOption {
		customAgent, err := promptCustomAgent()
		if err != nil {
			return nil, err
		}

		updatedAgents, err := agent.Upsert(cfg.Agents, customAgent)
		if err != nil {
			return nil, err
		}

		cfg.Agents = updatedAgents
		agentName = customAgent.Name
	}

	displayedPath := absolutePath
	if displayedPath == "" {
		displayedPath = "(agent-managed)"
	}
	displayedAlias := trimmedAlias
	if displayedAlias == "" {
		displayedAlias = "(none)"
	}
	displayedDescription := trimmedDescription
	if displayedDescription == "" {
		displayedDescription = "(none)"
	}
	summary := panelStyle.Render(fmt.Sprintf(
		"Name:        %s\nAlias:       %s\nAgent:       %s\nPath:        %s\nDescription: %s",
		trimmedName, displayedAlias, agentName, displayedPath, displayedDescription,
	))
	var confirm bool
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Save changes").
				Description(summary).
				Affirmative("Save").
				Negative("Cancel").
				Value(&confirm),
		),
	)
	if err := confirmForm.Run(); err != nil {
		return nil, err
	}
	if !confirm {
		return nil, nil
	}

	if absolutePath != "" {
		if err := os.MkdirAll(absolutePath, 0o755); err != nil {
			return nil, fmt.Errorf("create config directory %q: %w", absolutePath, err)
		}
	}

	updated := profile.Profile{
		Name:        trimmedName,
		Alias:       trimmedAlias,
		Path:        absolutePath,
		AgentName:   agentName,
		Description: trimmedDescription,
		CreatedAt:   original.CreatedAt,
	}
	cfg.Profiles[index] = updated

	if err := cfg.Save(); err != nil {
		return nil, err
	}

	return &updated, nil
}

func profilesExcluding(profiles []profile.Profile, index int) []profile.Profile {
	if index < 0 || index >= len(profiles) {
		return append([]profile.Profile(nil), profiles...)
	}
	others := make([]profile.Profile, 0, len(profiles)-1)
	others = append(others, profiles[:index]...)
	others = append(others, profiles[index+1:]...)
	return others
}

