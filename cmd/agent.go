package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/victorseara/aipim/internal/agent"
	"github.com/victorseara/aipim/internal/config"
)

var (
	agentAddName   string
	agentAddCmd    string

	agentCmd = &cobra.Command{
		Use:   "agent",
		Short: "Manage registered agents",
		Long:  "Manage the registry of AI agents available to aipim.\n\nSubcommands: list, add, rm, default.",
	}

	agentListCmd = &cobra.Command{
		Use:   "list",
		Short: "List registered agents",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return loadErrToExit(err)
			}

			if globalJSON {
				encodeJSON(struct {
					DefaultAgent string         `json:"default_agent"`
					Agents       []agent.Agent  `json:"agents"`
				}{
					DefaultAgent: cfg.DefaultAgentName,
					Agents:       cfg.Agents,
				})
				return nil
			}

			if globalQuiet {
				for _, a := range cfg.Agents {
					fmt.Println(a.Name)
				}
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tCOMMAND\tBUILT-IN\tDEFAULT")
			for _, a := range cfg.Agents {
				builtIn := "no"
				if a.IsBuiltIn {
					builtIn = "yes"
				}
				def := ""
				if strings.EqualFold(strings.TrimSpace(a.Name), strings.TrimSpace(cfg.DefaultAgentName)) {
					def = "*"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.Name, a.LaunchCmd, builtIn, def)
			}
			return w.Flush()
		},
	}

	agentAddCmdCmd = &cobra.Command{
		Use:   "add",
		Short: "Register a custom agent",
		Long: "Register a custom agent by name and launch command.\n\n" +
			"Example: aipim agent add --name \"My Agent\" --cmd \"/usr/local/bin/my-agent --flag\"",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(agentAddName)
			launchCmd := strings.TrimSpace(agentAddCmd)
			if name == "" {
				return usageErrorf("--name is required")
			}
			if launchCmd == "" {
				return usageErrorf("--cmd is required")
			}

			cfg, err := config.Load()
			if err != nil {
				return loadErrToExit(err)
			}

			updated, err := agent.Upsert(cfg.Agents, agent.Agent{Name: name, LaunchCmd: launchCmd})
			if err != nil {
				return configErrorf("%w", err)
			}
			cfg.Agents = updated

			if err := cfg.Save(); err != nil {
				return configErrorf("save config: %w", err)
			}

			if globalJSON {
				encodeJSON(map[string]string{"added": name, "cmd": launchCmd})
			} else if !globalQuiet {
				fmt.Fprintf(os.Stdout, "Registered agent %q (%s).\n", name, launchCmd)
			}
			return nil
		},
	}

	agentRmCmd = &cobra.Command{
		Use:     "rm <name>",
		Aliases: []string{"remove", "delete"},
		Short:   "Remove a custom agent",
		Long:    "Remove a custom agent. Built-in agents cannot be removed.",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return loadErrToExit(err)
			}

			name := strings.TrimSpace(args[0])

			if strings.EqualFold(strings.TrimSpace(cfg.DefaultAgentName), name) {
				return usageErrorf("agent %q is the current default agent. Run `aipim agent default <other>` first", name)
			}
			for _, p := range cfg.Profiles {
				if strings.EqualFold(strings.TrimSpace(p.AgentName), name) {
					return usageErrorf("agent %q is still used by profile %q. Reassign that profile first", name, p.Name)
				}
			}

			updated, err := agent.DeleteCustomByName(cfg.Agents, name)
			if err != nil {
				return configErrorf("%w", err)
			}
			cfg.Agents = updated

			if err := cfg.Save(); err != nil {
				return configErrorf("save config: %w", err)
			}

			if globalJSON {
				encodeJSON(map[string]string{"removed": name})
			} else if !globalQuiet {
				fmt.Fprintf(os.Stdout, "Removed agent %q.\n", name)
			}
			return nil
		},
	}

	agentDefaultCmd = &cobra.Command{
		Use:   "default <name>",
		Short: "Set the default agent",
		Long:  "Set the agent used by profiles that do not specify one.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return loadErrToExit(err)
			}

			name := strings.TrimSpace(args[0])
			if _, ok := agent.FindByName(cfg.Agents, name); !ok {
				return configErrorf("agent %q is not registered. Run `aipim agent list` to see options", name)
			}
			cfg.DefaultAgentName = name

			if err := cfg.Save(); err != nil {
				return configErrorf("save config: %w", err)
			}

			if globalJSON {
				encodeJSON(map[string]string{"default_agent": name})
			} else if !globalQuiet {
				fmt.Fprintf(os.Stdout, "Default agent set to %q.\n", name)
			}
			return nil
		},
	}
)

func init() {
	agentAddCmdCmd.Flags().StringVar(&agentAddName, "name", "", "Display name for the custom agent")
	agentAddCmdCmd.Flags().StringVar(&agentAddCmd, "cmd", "", "Launch command (e.g. `/usr/local/bin/my-agent --flag`)")

	agentCmd.AddCommand(agentListCmd, agentAddCmdCmd, agentRmCmd, agentDefaultCmd)
	rootCmd.AddCommand(agentCmd)
}
