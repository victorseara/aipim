package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/google/shlex"
	"github.com/spf13/cobra"

	"github.com/victorseara/aipim/internal/agent"
	"github.com/victorseara/aipim/internal/config"
	"github.com/victorseara/aipim/internal/profile"
	"github.com/victorseara/aipim/internal/tui"
)

var (
	launchProfileName string
	launchMessage     string

	launchCmd = &cobra.Command{
		Use:   "launch [-- <agent args>...]",
		Short: "Launch an agent with an isolated profile",
		Long: "Launch an agent with an isolated profile.\n\n" +
			"Anything after `--` is forwarded verbatim to the agent. " +
			"Example: `aipim launch -p sgws -- -p \"my prompt\"`.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfigWithRecovery(true)
			if err != nil {
				return err
			}

			return launchProfile(cfg, launchProfileName, launchMessage, args)
		},
	}
)

func init() {
	launchCmd.Flags().StringVarP(&launchProfileName, "profile", "p", "", "Profile name or alias to launch")
	launchCmd.Flags().StringVarP(&launchMessage, "message", "m", "", "Initial message to pass to the agent")
	_ = launchCmd.RegisterFlagCompletionFunc("profile", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return profileCompletions(toComplete), cobra.ShellCompDirectiveNoFileComp
	})
}

func launchProfile(cfg *config.AppConfig, requestedProfile, message string, extraArgs []string) error {
	selectedProfile, err := selectProfile(cfg, requestedProfile)
	if err != nil {
		return err
	}

	selectedProfile.Path, err = ensureAbsolutePath(selectedProfile.Path)
	if err != nil {
		return configErrorf("resolve profile path: %w", err)
	}

	agentName := strings.TrimSpace(selectedProfile.AgentName)
	if agentName == "" {
		agentName = strings.TrimSpace(cfg.DefaultAgentName)
	}
	if agentName == "" {
		if !interactive() {
			return configErrorf("profile %q has no agent assigned and no default agent is set. Run `aipim edit %s --agent <name>` to fix", selectedProfile.Name, selectedProfile.Name)
		}
		if err := tui.RunOnboarding(cfg, false); err != nil {
			return err
		}
		agentName = strings.TrimSpace(cfg.DefaultAgentName)
	}

	selectedAgent, ok := agent.FindByName(cfg.Agents, agentName)
	if !ok {
		return configErrorf("agent %q is not registered. Run `aipim agent list` to see available agents", agentName)
	}

	if selectedProfile.Path != "" {
		if err := os.MkdirAll(selectedProfile.Path, 0o755); err != nil {
			return configErrorf("create profile config directory %q: %w", selectedProfile.Path, err)
		}
	}

	return execAgent(selectedAgent, selectedProfile, message, extraArgs)
}

func selectProfile(cfg *config.AppConfig, requestedProfile string) (profile.Profile, error) {
	if strings.TrimSpace(requestedProfile) != "" {
		found, _, ok := profile.FindByIdentifier(cfg.Profiles, requestedProfile)
		if !ok {
			available := profileIdentifiers(cfg.Profiles)
			if len(available) == 0 {
				return profile.Profile{}, usageErrorf("profile %q does not exist and no profiles are configured. Run `aipim create` to add one", requestedProfile)
			}

			return profile.Profile{}, usageErrorf(
				"profile %q does not exist. Available: %s. Run `aipim list` for details",
				requestedProfile,
				strings.Join(available, ", "),
			)
		}

		return found, nil
	}

	switch len(cfg.Profiles) {
	case 0:
		if !interactive() {
			return profile.Profile{}, configErrorf("no profiles configured. Run `aipim create --name <n> --agent <a>` to add one")
		}
		created, err := tui.RunCreateFlow(cfg)
		if err != nil {
			return profile.Profile{}, err
		}
		if created == nil {
			return profile.Profile{}, cancelledError
		}

		return *created, nil
	case 1:
		return cfg.Profiles[0], nil
	default:
		if !interactive() {
			return profile.Profile{}, usageErrorf("multiple profiles configured but none specified. Pass `--profile <name>` or pick one from `aipim list`")
		}
		selected, err := tui.SelectProfile(cfg.Profiles)
		if err != nil {
			return profile.Profile{}, err
		}
		return selected, nil
	}
}

func execAgent(selectedAgent agent.Agent, selectedProfile profile.Profile, message string, extraArgs []string) error {
	parts, err := shlex.Split(strings.TrimSpace(selectedAgent.LaunchCmd))
	if err != nil {
		return configErrorf("parse launch command for agent %q: %w", selectedAgent.Name, err)
	}
	if len(parts) == 0 {
		return configErrorf("agent %q has an empty launch command", selectedAgent.Name)
	}

	binaryPath, err := exec.LookPath(parts[0])
	if err != nil {
		return agentNotFoundErrorf(
			"agent binary %q not found in PATH. Install it first, or run `aipim agent add` to register a different binary",
			parts[0],
		)
	}

	args := append([]string{filepath.Base(binaryPath)}, parts[1:]...)
	args = append(args, extraArgs...)
	if strings.TrimSpace(message) != "" {
		args = append(args, message)
	}

	env := os.Environ()
	if selectedProfile.Path != "" {
		envVarName := agent.ConfigEnvVar(selectedAgent.LaunchCmd)
		env = append(env, fmt.Sprintf("%s=%s", envVarName, selectedProfile.Path))
	}

	if err := syscall.Exec(binaryPath, args, env); err != nil {
		return withCode(ExitGeneric, fmt.Errorf("exec %q: %w", binaryPath, err))
	}
	return nil
}

func profileIdentifiers(profiles []profile.Profile) []string {
	identifiers := make([]string, 0, len(profiles))
	for _, p := range profiles {
		if strings.TrimSpace(p.Alias) != "" {
			identifiers = append(identifiers, fmt.Sprintf("%s (%s)", p.Name, strings.TrimSpace(p.Alias)))
			continue
		}
		identifiers = append(identifiers, p.Name)
	}
	return identifiers
}

func ensureAbsolutePath(path string) (string, error) {
	expanded, err := config.ExpandPath(path)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(expanded) == "" {
		return "", nil
	}

	if filepath.IsAbs(expanded) {
		return expanded, nil
	}

	absolute, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path %q: %w", expanded, err)
	}

	return absolute, nil
}
