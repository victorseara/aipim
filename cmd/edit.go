package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/victorseara/aipim/internal/agent"
	"github.com/victorseara/aipim/internal/config"
	"github.com/victorseara/aipim/internal/profile"
	"github.com/victorseara/aipim/internal/tui"
)

var (
	editAlias       string
	editAgent       string
	editPath        string
	editDescription string
	editClearAlias  bool

	editAliasSet       bool
	editAgentSet       bool
	editPathSet        bool
	editDescriptionSet bool

	editCmd = &cobra.Command{
		Use:   "edit <name|alias>",
		Short: "Edit a profile (CLI flags or TUI)",
		Long: "Edit a profile in place.\n\n" +
			"With no flags, opens the interactive TUI edit form.\n" +
			"With one or more --alias / --agent / --path / --description flags, applies the patch non-interactively.\n" +
			"Pass `--description -` to read the description from stdin (useful for long, scripted descriptions).",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return profileCompletions(toComplete), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			editAliasSet = cmd.Flags().Changed("alias") || cmd.Flags().Changed("clear-alias")
			editAgentSet = cmd.Flags().Changed("agent")
			editPathSet = cmd.Flags().Changed("path")
			editDescriptionSet = cmd.Flags().Changed("description")

			cfg, err := config.Load()
			if err != nil {
				return loadErrToExit(err)
			}

			anyFlag := editAliasSet || editAgentSet || editPathSet || editDescriptionSet
			if !anyFlag {
				if !interactive() {
					return usageErrorf("no patch flags provided and no TTY available. Pass --alias, --agent, --path, or --description")
				}
				_, err := tui.RunEditFlow(cfg, args[0])
				return err
			}

			return editProfilePatch(cfg, args[0])
		},
	}
)

func init() {
	editCmd.Flags().StringVar(&editAlias, "alias", "", "Set the profile alias")
	editCmd.Flags().BoolVar(&editClearAlias, "clear-alias", false, "Remove the existing alias")
	editCmd.Flags().StringVar(&editAgent, "agent", "", "Set the assigned agent")
	editCmd.Flags().StringVar(&editPath, "path", "", "Set the profile config directory path")
	editCmd.Flags().StringVar(&editDescription, "description", "", "Set the agent-selection description. Pass `-` to read from stdin")
	rootCmd.AddCommand(editCmd)
}

func editProfilePatch(cfg *config.AppConfig, identifier string) error {
	found, index, ok := profile.FindByIdentifier(cfg.Profiles, identifier)
	if !ok {
		return usageErrorf("profile %q does not exist. Run `aipim list` to see available profiles", identifier)
	}

	others := append([]profile.Profile(nil), cfg.Profiles[:index]...)
	others = append(others, cfg.Profiles[index+1:]...)

	if editAliasSet {
		if editClearAlias {
			found.Alias = ""
		} else {
			if err := profile.ValidateAlias(editAlias, found.Name, others); err != nil {
				return usageErrorf("%w", err)
			}
			found.Alias = strings.TrimSpace(editAlias)
		}
	}

	if editAgentSet {
		agentName := strings.TrimSpace(editAgent)
		if _, ok := agent.FindByName(cfg.Agents, agentName); !ok {
			return configErrorf("agent %q is not registered. Run `aipim agent list` to see options", agentName)
		}
		found.AgentName = agentName
	}

	if editPathSet {
		expanded, err := config.ExpandPath(editPath)
		if err != nil {
			return configErrorf("expand path: %w", err)
		}
		if expanded != "" {
			absolute, err := filepath.Abs(expanded)
			if err != nil {
				return configErrorf("resolve absolute path %q: %w", expanded, err)
			}
			expanded = absolute
		}
		found.Path = expanded
	}

	if editDescriptionSet {
		descValue, err := resolveDescription(editDescription)
		if err != nil {
			return err
		}
		found.Description = strings.TrimSpace(descValue)
	}

	cfg.Profiles[index] = found
	if err := cfg.Save(); err != nil {
		return configErrorf("save config: %w", err)
	}

	if globalJSON {
		printProfileJSON(found)
	} else {
		fmt.Fprintf(os.Stdout, "Updated profile %q.\n", found.Name)
	}
	return nil
}

// resolveDescription returns the description string, reading from stdin when the value is `-`.
func resolveDescription(value string) (string, error) {
	if value != "-" {
		return value, nil
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", withCode(ExitGeneric, fmt.Errorf("read description from stdin: %w", err))
	}
	return string(data), nil
}

func printProfileJSON(p profile.Profile) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(p)
}
