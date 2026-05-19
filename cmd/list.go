package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/victorseara/aipim/internal/config"
	"github.com/victorseara/aipim/internal/profile"
)

var (
	listWithDescription bool

	listCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List configured profiles",
		Long: "List configured profiles.\n\n" +
			"Default output is a human-readable table.\n" +
			"Use --json for machine-readable output (recommended for AI agents and scripts).\n" +
			"Use --quiet to emit only profile names, one per line, suitable for piping into xargs.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return loadErrToExit(err)
			}

			switch {
			case globalJSON:
				return printProfilesJSON(cfg)
			case globalQuiet:
				return printProfilesQuiet(cfg)
			default:
				return printProfilesTable(cfg, listWithDescription)
			}
		},
	}
)

func init() {
	listCmd.Flags().BoolVar(&listWithDescription, "with-description", false, "Include a truncated description column in the default table output (ignored with --json/--quiet)")
	rootCmd.AddCommand(listCmd)
}

// loadErrToExit converts a config.Load error into a structured ExitError.
// Useful for read-only commands that should not trigger the interactive onboarding flow.
func loadErrToExit(err error) error {
	return configErrorf("load config: %w. Run `aipim` in a terminal to set up your first profile, or `aipim create --no-tui --name ... --agent ...`", err)
}

func printProfilesJSON(cfg *config.AppConfig) error {
	payload := struct {
		DefaultAgent string            `json:"default_agent"`
		Profiles     []profile.Profile `json:"profiles"`
	}{
		DefaultAgent: cfg.DefaultAgentName,
		Profiles:     cfg.Profiles,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		return withCode(ExitGeneric, fmt.Errorf("encode json: %w", err))
	}
	return nil
}

func printProfilesQuiet(cfg *config.AppConfig) error {
	for _, p := range cfg.Profiles {
		fmt.Println(p.Name)
	}
	return nil
}

func printProfilesTable(cfg *config.AppConfig, withDescription bool) error {
	if len(cfg.Profiles) == 0 {
		fmt.Println("No profiles configured. Run `aipim create` to add one.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	header := "NAME\tALIAS\tAGENT\tPATH"
	if withDescription {
		header += "\tDESCRIPTION"
	}
	fmt.Fprintln(w, header)

	for _, p := range cfg.Profiles {
		alias := strings.TrimSpace(p.Alias)
		if alias == "" {
			alias = "-"
		}
		agentName := strings.TrimSpace(p.AgentName)
		if agentName == "" {
			agentName = cfg.DefaultAgentName
		}
		if agentName == "" {
			agentName = "-"
		}
		path := strings.TrimSpace(p.Path)
		if path == "" {
			path = "(agent-managed)"
		}

		row := fmt.Sprintf("%s\t%s\t%s\t%s", p.Name, alias, agentName, path)
		if withDescription {
			row += "\t" + truncateOneLine(p.Description, 60)
		}
		fmt.Fprintln(w, row)
	}
	return w.Flush()
}

func truncateOneLine(s string, max int) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return "-"
	}
	// First line only.
	if idx := strings.IndexByte(trimmed, '\n'); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	if len(trimmed) <= max {
		return trimmed
	}
	if max < 1 {
		return ""
	}
	return trimmed[:max-1] + "…"
}
