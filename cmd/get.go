package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/victorseara/aipim/internal/config"
	"github.com/victorseara/aipim/internal/profile"
)

var getCmd = &cobra.Command{
	Use:     "get <name|alias>",
	Aliases: []string{"show"},
	Short:   "Show details for a single profile",
	Long: "Show all stored fields for a single profile, including its description.\n\n" +
		"Lookup is by name first, then by alias.\n" +
		"Use --json for machine-readable output (recommended for AI agents).",
	Args: cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return profileCompletions(toComplete), cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return loadErrToExit(err)
		}

		found, _, ok := profile.FindByIdentifier(cfg.Profiles, args[0])
		if !ok {
			return usageErrorf("profile %q does not exist. Run `aipim list` to see available profiles", args[0])
		}

		if globalJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(found); err != nil {
				return withCode(ExitGeneric, fmt.Errorf("encode json: %w", err))
			}
			return nil
		}

		printProfileHuman(found, cfg.DefaultAgentName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}

func printProfileHuman(p profile.Profile, defaultAgent string) {
	agentName := strings.TrimSpace(p.AgentName)
	if agentName == "" {
		agentName = defaultAgent + " (default)"
	}
	alias := strings.TrimSpace(p.Alias)
	if alias == "" {
		alias = "-"
	}
	path := strings.TrimSpace(p.Path)
	if path == "" {
		path = "(agent-managed)"
	}

	fmt.Printf("Name:        %s\n", p.Name)
	fmt.Printf("Alias:       %s\n", alias)
	fmt.Printf("Agent:       %s\n", agentName)
	fmt.Printf("Path:        %s\n", path)
	fmt.Printf("Created:     %s\n", p.CreatedAt)

	description := strings.TrimSpace(p.Description)
	if description == "" {
		fmt.Println("Description: (none)")
		return
	}
	fmt.Println("Description:")
	for _, line := range strings.Split(description, "\n") {
		fmt.Printf("  │ %s\n", line)
	}
}

// profileCompletions returns profile names + aliases matching toComplete prefix.
// Safe to call when the config can't be loaded; returns an empty slice in that case.
func profileCompletions(toComplete string) []string {
	cfg, err := config.Load()
	if err != nil {
		return nil
	}
	suggestions := make([]string, 0, len(cfg.Profiles)*2)
	prefix := strings.ToLower(toComplete)
	for _, p := range cfg.Profiles {
		if strings.HasPrefix(strings.ToLower(p.Name), prefix) {
			suggestions = append(suggestions, p.Name)
		}
		if alias := strings.TrimSpace(p.Alias); alias != "" && strings.HasPrefix(strings.ToLower(alias), prefix) {
			suggestions = append(suggestions, alias)
		}
	}
	return suggestions
}
