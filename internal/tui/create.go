package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/huh"

	"github.com/victorseara/aipim/internal/agent"
	"github.com/victorseara/aipim/internal/config"
	"github.com/victorseara/aipim/internal/profile"
)

const customAgentOption = "__custom__"

// RunOnboarding runs the first-run onboarding flow and saves the config.
func RunOnboarding(cfg *config.AppConfig, showWelcome bool) error {
	if showWelcome {
		var proceed bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Welcome to aipim").
					Description(panelStyle.Render("Create isolated AI agent profiles, keep configs separate, and switch instantly.")).
					Affirmative("Continue").
					Negative("Quit").
					Value(&proceed),
			),
		)
		if err := form.Run(); err != nil {
			return err
		}
		if !proceed {
			return errors.New("onboarding cancelled")
		}
	}

	selectedAgentName, err := selectAgentName("Choose your default agent", "Set the agent aipim should use when a profile does not specify one.", cfg.Agents, true, cfg.DefaultAgentName)
	if err != nil {
		return err
	}

	if selectedAgentName == customAgentOption {
		customAgent, err := promptCustomAgent()
		if err != nil {
			return err
		}

		updatedAgents, err := agent.Upsert(cfg.Agents, customAgent)
		if err != nil {
			return err
		}

		cfg.Agents = updatedAgents
		cfg.DefaultAgentName = customAgent.Name
	} else {
		cfg.DefaultAgentName = selectedAgentName
	}

	return cfg.Save()
}

// RunCreateFlow runs the interactive profile creation flow and saves the config.
func RunCreateFlow(cfg *config.AppConfig) (*profile.Profile, error) {
	var name string
	nameForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Profile name").
				Description("Use a short, memorable name with no slashes.").
				Value(&name).
				Validate(func(value string) error {
					return profile.ValidateName(value, cfg.Profiles)
				}),
		),
	)
	if err := nameForm.Run(); err != nil {
		return nil, err
	}

	trimmedName := strings.TrimSpace(name)

	var alias string
	aliasForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Profile alias").
				Description("Optional shortcut for `aipim <alias>`. Leave empty to skip.").
				Value(&alias).
				Validate(func(value string) error {
					return profile.ValidateAlias(value, trimmedName, cfg.Profiles)
				}),
		),
	)
	if err := aliasForm.Run(); err != nil {
		return nil, err
	}
	trimmedAlias := strings.TrimSpace(alias)

	defaultPath, err := config.DefaultProfilePath(trimmedName)
	if err != nil {
		return nil, err
	}

	path := defaultPath
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

	var description string
	descriptionForm := huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("Description").
				Description("Optional. Helps AI agents pick this profile from a user's prompt.\nExamples: \"SGWS client work, Snowflake MCP, React + Tailwind\".").
				Lines(4).
				Value(&description),
		),
	)
	if err := descriptionForm.Run(); err != nil {
		return nil, err
	}
	trimmedDescription := strings.TrimSpace(description)

	selectedAgentName, err := selectAgentName("Select an agent", "Pick a registered agent or create a custom command for this profile.", cfg.Agents, false, cfg.DefaultAgentName)
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
	summary := panelStyle.Render(fmt.Sprintf("Name: %s\nAlias: %s\nAgent: %s\nPath: %s\nDescription: %s", trimmedName, displayedAlias, agentName, displayedPath, displayedDescription))
	var confirm bool
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Create profile").
				Description(summary).
				Affirmative("Save profile").
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

	created := profile.Profile{
		Name:        trimmedName,
		Alias:       trimmedAlias,
		Path:        absolutePath,
		AgentName:   agentName,
		Description: trimmedDescription,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	updatedProfiles, err := profile.Add(cfg.Profiles, created)
	if err != nil {
		return nil, err
	}

	cfg.Profiles = updatedProfiles
	if err := cfg.Save(); err != nil {
		return nil, err
	}

	return &created, nil
}

// RunSettingsFlow runs the settings flow for agent management and default agent selection.
func RunSettingsFlow(cfg *config.AppConfig) error {
	choices := []huh.Option[string]{
		huh.NewOption("Set default agent", "default"),
		huh.NewOption("Register custom agent", "register"),
	}

	customAgents := make([]agent.Agent, 0)
	for _, configuredAgent := range cfg.Agents {
		if !configuredAgent.IsBuiltIn {
			customAgents = append(customAgents, configuredAgent)
		}
	}
	if len(customAgents) > 0 {
		choices = append(choices, huh.NewOption("Remove custom agent", "remove"))
	}
	choices = append(choices, huh.NewOption("Back", "back"))

	var action string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Settings").
				Description("Manage your default agent and custom agent registry.").
				Options(choices...).
				Value(&action),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}

	switch action {
	case "default":
		selectedAgentName, err := selectAgentName("Default agent", "Profiles without a dedicated agent will use this default.", cfg.Agents, true, cfg.DefaultAgentName)
		if err != nil {
			return err
		}

		if selectedAgentName == customAgentOption {
			customAgent, err := promptCustomAgent()
			if err != nil {
				return err
			}

			updatedAgents, err := agent.Upsert(cfg.Agents, customAgent)
			if err != nil {
				return err
			}

			cfg.Agents = updatedAgents
			cfg.DefaultAgentName = customAgent.Name
		} else {
			cfg.DefaultAgentName = selectedAgentName
		}

		return cfg.Save()
	case "register":
		customAgent, err := promptCustomAgent()
		if err != nil {
			return err
		}

		updatedAgents, err := agent.Upsert(cfg.Agents, customAgent)
		if err != nil {
			return err
		}

		cfg.Agents = updatedAgents
		return cfg.Save()
	case "remove":
		var selectedAgentName string
		options := make([]huh.Option[string], 0, len(customAgents))
		for _, configuredAgent := range customAgents {
			options = append(options, huh.NewOption(configuredAgent.Name+"  ("+configuredAgent.LaunchCmd+")", configuredAgent.Name))
		}

		removeForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Remove custom agent").
					Description("Built-in agents cannot be deleted.").
					Options(options...).
					Value(&selectedAgentName),
			),
		)
		if err := removeForm.Run(); err != nil {
			return err
		}

		if strings.EqualFold(strings.TrimSpace(cfg.DefaultAgentName), strings.TrimSpace(selectedAgentName)) {
			return fmt.Errorf("agent %q is the current default agent; choose another default first", selectedAgentName)
		}
		for _, configuredProfile := range cfg.Profiles {
			if strings.EqualFold(strings.TrimSpace(configuredProfile.AgentName), strings.TrimSpace(selectedAgentName)) {
				return fmt.Errorf("agent %q is still used by profile %q", selectedAgentName, configuredProfile.Name)
			}
		}

		updatedAgents, err := agent.DeleteCustomByName(cfg.Agents, selectedAgentName)
		if err != nil {
			return err
		}
		cfg.Agents = updatedAgents
		return cfg.Save()
	default:
		return nil
	}
}

// SelectProfile prompts the user to choose a configured profile.
func SelectProfile(profiles []profile.Profile) (profile.Profile, error) {
	options := make([]huh.Option[string], 0, len(profiles))
	for _, configuredProfile := range profiles {
		options = append(options, huh.NewOption(
			profileLabel(configuredProfile),
			configuredProfile.Name,
		))
	}

	var selectedName string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose a profile").
				Description("Select the profile to launch.").
				Options(options...).
				Value(&selectedName),
		),
	)
	if err := form.Run(); err != nil {
		return profile.Profile{}, err
	}

	selectedProfile, _, ok := profile.FindByName(profiles, selectedName)
	if !ok {
		return profile.Profile{}, fmt.Errorf("profile %q does not exist", selectedName)
	}

	return selectedProfile, nil
}

// ConfirmConfigReset offers to reset a corrupted config file.
func ConfirmConfigReset(loadErr error) (bool, error) {
	var reset bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Config recovery").
				Description(dangerStyle.Render("Your aipim config is unreadable.") + "\n\n" + mutedStyle.Render(loadErr.Error()) + "\n\nReset it and rebuild from onboarding?").
				Affirmative("Reset config").
				Negative("Cancel").
				Value(&reset),
		),
	)
	if err := form.Run(); err != nil {
		return false, err
	}

	return reset, nil
}

func selectAgentName(title, description string, agents []agent.Agent, allowCustom bool, current string) (string, error) {
	options := make([]huh.Option[string], 0, len(agents)+1)
	for _, configuredAgent := range agents {
		options = append(options, huh.NewOption(configuredAgent.Name+"  ("+configuredAgent.LaunchCmd+")", configuredAgent.Name))
	}
	if allowCustom {
		options = append(options, huh.NewOption("Register custom agent…", customAgentOption))
	} else {
		options = append(options, huh.NewOption("Custom command…", customAgentOption))
	}

	selected := strings.TrimSpace(current)
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(title).
				Description(description).
				Options(options...).
				Value(&selected),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}

	return selected, nil
}

func promptCustomAgent() (agent.Agent, error) {
	var (
		name      string
		launchCmd string
	)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Agent name").
				Description("This is the label shown in aipim.").
				Value(&name).
				Validate(func(value string) error {
					if strings.TrimSpace(value) == "" {
						return errors.New("agent name cannot be empty")
					}

					return nil
				}),
			huh.NewInput().
				Title("Launch command").
				Description("Examples: claude, codex, gh copilot, /usr/local/bin/custom-agent --fast").
				Value(&launchCmd).
				Validate(func(value string) error {
					if strings.TrimSpace(value) == "" {
						return errors.New("launch command cannot be empty")
					}

					return nil
				}),
		),
	)
	if err := form.Run(); err != nil {
		return agent.Agent{}, err
	}

	return agent.Agent{
		Name:      strings.TrimSpace(name),
		LaunchCmd: strings.TrimSpace(launchCmd),
		IsBuiltIn: false,
	}, nil
}

func profileLabel(configuredProfile profile.Profile) string {
	path := strings.TrimSpace(configuredProfile.Path)
	if path == "" {
		path = "agent-managed"
	} else if home, err := os.UserHomeDir(); err == nil {
		prefix := home + string(filepath.Separator)
		if strings.HasPrefix(path, prefix) {
			path = "~/" + strings.TrimPrefix(path, prefix)
		}
	}

	name := configuredProfile.Name
	if alias := strings.TrimSpace(configuredProfile.Alias); alias != "" {
		name = fmt.Sprintf("%s (%s)", configuredProfile.Name, alias)
	}

	if configuredProfile.AgentName == "" {
		return fmt.Sprintf("%s  (%s)", name, path)
	}

	return fmt.Sprintf("%s  [%s]  (%s)", name, configuredProfile.AgentName, path)
}
