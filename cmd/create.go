package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/victorseara/aipim/internal/agent"
	"github.com/victorseara/aipim/internal/config"
	"github.com/victorseara/aipim/internal/profile"
	"github.com/victorseara/aipim/internal/tui"
)

var (
	createName         string
	createAlias        string
	createAgent        string
	createPath         string
	createDescription  string
	createSetAsDefault bool

	createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a profile",
		Long: "Create a profile.\n\n" +
			"With no flags, opens the interactive TUI creation flow.\n" +
			"With --name and --agent, creates a profile non-interactively (useful for scripts and AI agents).\n" +
			"Pass `--description -` to read the description from stdin.\n" +
			"Use --no-tui (global flag) to enforce non-interactive mode and error out if any required field is missing.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			anyFlag := cmd.Flags().Changed("name") ||
				cmd.Flags().Changed("alias") ||
				cmd.Flags().Changed("agent") ||
				cmd.Flags().Changed("path") ||
				cmd.Flags().Changed("description") ||
				cmd.Flags().Changed("set-default")

			forceNonInteractive := anyFlag || globalNoTUI || globalJSON

			cfg, err := loadConfigForCreate(forceNonInteractive)
			if err != nil {
				return err
			}

			if forceNonInteractive || !interactive() {
				return createProfileFromFlags(cmd, cfg)
			}

			created, err := tui.RunCreateFlow(cfg)
			if err != nil {
				return err
			}
			if created == nil {
				return cancelledError
			}

			launchID := created.Name
			if alias := strings.TrimSpace(created.Alias); alias != "" {
				launchID = alias
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created profile %q.\nLaunch it with: aipim %s\n", created.Name, launchID)
			return nil
		},
	}
)

func init() {
	createCmd.Flags().StringVar(&createName, "name", "", "Profile name (required for non-interactive create)")
	createCmd.Flags().StringVar(&createAlias, "alias", "", "Optional alias for `aipim <alias>` shortcut")
	createCmd.Flags().StringVar(&createAgent, "agent", "", "Agent to assign (required for non-interactive create)")
	createCmd.Flags().StringVar(&createPath, "path", "", "Profile config directory. Defaults to <config-dir>/profiles/<name>. Pass an empty value to leave the agent in charge of its own directory")
	createCmd.Flags().StringVar(&createDescription, "description", "", "Agent-selection description. Pass `-` to read from stdin")
	createCmd.Flags().BoolVar(&createSetAsDefault, "set-default", false, "Also set this profile's agent as the default agent")
}

// loadConfigForCreate loads the config. When forced non-interactive, missing config
// is fatal (no TUI onboarding) — we bootstrap an empty config with built-in agents instead.
func loadConfigForCreate(forceNonInteractive bool) (*config.AppConfig, error) {
	if !forceNonInteractive {
		return loadConfigWithRecovery(true)
	}

	cfg, err := config.Load()
	if err == nil {
		return cfg, nil
	}
	if errors.Is(err, config.ErrConfigNotFound) {
		return config.DefaultConfig(), nil
	}
	return nil, configErrorf("load config: %w", err)
}

func createProfileFromFlags(cmd *cobra.Command, cfg *config.AppConfig) error {
	name := strings.TrimSpace(createName)
	if name == "" {
		return usageErrorf("--name is required for non-interactive create")
	}
	agentName := strings.TrimSpace(createAgent)
	if agentName == "" {
		agentName = strings.TrimSpace(cfg.DefaultAgentName)
	}
	if agentName == "" {
		return usageErrorf("--agent is required when no default agent is configured")
	}

	if _, ok := agent.FindByName(cfg.Agents, agentName); !ok {
		return configErrorf("agent %q is not registered. Run `aipim agent list` for options", agentName)
	}

	if err := profile.ValidateName(name, cfg.Profiles); err != nil {
		return usageErrorf("%w", err)
	}
	if err := profile.ValidateAlias(createAlias, name, cfg.Profiles); err != nil {
		return usageErrorf("%w", err)
	}

	absolutePath, err := resolveCreatePath(cmd, name)
	if err != nil {
		return err
	}

	description, err := resolveDescription(createDescription)
	if err != nil {
		return err
	}

	created := profile.Profile{
		Name:        name,
		Alias:       strings.TrimSpace(createAlias),
		Path:        absolutePath,
		AgentName:   agentName,
		Description: strings.TrimSpace(description),
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	updated, err := profile.Add(cfg.Profiles, created)
	if err != nil {
		return usageErrorf("%w", err)
	}
	cfg.Profiles = updated

	if createSetAsDefault || strings.TrimSpace(cfg.DefaultAgentName) == "" {
		cfg.DefaultAgentName = agentName
	}

	if absolutePath != "" {
		if err := os.MkdirAll(absolutePath, 0o755); err != nil {
			return configErrorf("create config directory %q: %w", absolutePath, err)
		}
	}

	if err := cfg.Save(); err != nil {
		return configErrorf("save config: %w", err)
	}

	if globalJSON {
		printProfileJSON(created)
		return nil
	}

	if !globalQuiet {
		launchID := created.Name
		if created.Alias != "" {
			launchID = created.Alias
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Created profile %q.\nLaunch it with: aipim %s\n", created.Name, launchID)
	}
	return nil
}

func resolveCreatePath(cmd *cobra.Command, name string) (string, error) {
	rawPath := createPath
	if !cmd.Flags().Changed("path") {
		def, err := config.DefaultProfilePath(name)
		if err != nil {
			return "", configErrorf("resolve default profile path: %w", err)
		}
		rawPath = def
	}
	expanded, err := config.ExpandPath(rawPath)
	if err != nil {
		return "", configErrorf("expand path: %w", err)
	}
	if strings.TrimSpace(expanded) == "" {
		return "", nil
	}
	absolute, err := filepath.Abs(expanded)
	if err != nil {
		return "", configErrorf("resolve absolute path %q: %w", expanded, err)
	}
	return absolute, nil
}
