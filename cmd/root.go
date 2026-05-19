package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/victorseara/aipim/internal/config"
	"github.com/victorseara/aipim/internal/tui"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "aipim",
	})

	rootMessage   string
	globalJSON    bool
	globalQuiet   bool
	globalNoTUI   bool
	globalConfDir string

	rootCmd = &cobra.Command{
		Use:   "aipim [profile|alias]",
		Short: "Manage isolated profiles for AI coding agents",
		Long:  "aipim keeps agent configuration directories isolated so you can switch between agent profiles instantly.\n\nRun `aipim` with no arguments to open the TUI, or `aipim <profile-or-alias>` to launch a profile directly.",
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return profileCompletions(toComplete), cobra.ShellCompDirectiveNoFileComp
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(globalConfDir) != "" {
				expanded, err := config.ExpandPath(globalConfDir)
				if err != nil {
					return configErrorf("resolve --config-dir: %w", err)
				}
				config.SetConfigDirOverride(expanded)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfigWithRecovery(true)
			if err != nil {
				return err
			}

			if len(args) == 1 {
				return launchProfile(cfg, args[0], rootMessage, nil)
			}

			if !interactive() {
				return usageErrorf("no profile specified and no TTY available; run `aipim list` or pass a profile name")
			}

			result, err := tui.RunApp(cfg)
			if err != nil {
				return err
			}

			if result.Action == tui.ActionLaunch {
				return launchProfile(cfg, result.ProfileName, rootMessage, nil)
			}

			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
)

const logo = `
      _       _
  __ _(_)_ __(_)_ __ ___
 / _  | |  __| |  _   _ \
| (_| | | |  | | | | | | |
 \__,_|_|_|  |_|_| |_| |_|
`

var (
	logoStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7C3AED"))
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
)

func init() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	rootCmd.PersistentFlags().BoolVar(&globalJSON, "json", false, "Emit machine-readable JSON output (where supported) and JSON error envelopes")
	rootCmd.PersistentFlags().BoolVar(&globalQuiet, "quiet", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVar(&globalNoTUI, "no-tui", false, "Never open the interactive TUI; fail with an error if interaction is required")
	rootCmd.PersistentFlags().StringVar(&globalConfDir, "config-dir", "", "Override the aipim config directory (also: AIPIM_CONFIG_HOME)")

	rootCmd.Flags().StringVarP(&rootMessage, "message", "m", "", "Initial message to pass to the agent")

	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(launchCmd)
	rootCmd.SetHelpTemplate(buildHelpTemplate())

	rootCmd.Version = versionString()
	rootCmd.SetVersionTemplate("aipim {{.Version}}\n")
}

// Execute runs the aipim CLI and returns the process exit code.
func Execute() int {
	err := rootCmd.Execute()
	if err == nil {
		return ExitOK
	}

	code := ExitCodeFor(err)
	emitError(err, code)
	return code
}

// emitError prints the error to stderr (plain text) or stdout (JSON envelope) depending on flags.
// Silent exits (commands that have already emitted their own structured output) are skipped.
func emitError(err error, code int) {
	if isSilentExit(err) {
		return
	}
	if globalJSON {
		payload := struct {
			Error string `json:"error"`
			Code  int    `json:"code"`
		}{
			Error: err.Error(),
			Code:  code,
		}
		enc := json.NewEncoder(os.Stdout)
		_ = enc.Encode(payload)
		return
	}
	if errors.Is(err, cancelledError) {
		return
	}
	fmt.Fprintln(os.Stderr, "Error:", err.Error())
}

// interactive returns true when both stdin and stderr are TTYs and --no-tui isn't set.
// Use this to decide whether to launch huh forms or fall back to a CLI-friendly error.
func interactive() bool {
	if globalNoTUI || globalJSON {
		return false
	}
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		return false
	}
	return true
}

func buildHelpTemplate() string {
	var builder strings.Builder

	builder.WriteString(logoStyle.Render(logo))
	builder.WriteString("\n")
	builder.WriteString(helpStyle.Render("AI Agent Profile Manager"))
	builder.WriteString("\n\n")
	builder.WriteString("Usage:\n  {{.UseLine}}\n\n")
	builder.WriteString("Commands:\n")
	builder.WriteString("{{range .Commands}}{{if (or .IsAvailableCommand .IsAdditionalHelpTopicCommand)}}")
	builder.WriteString("  {{rpad .Name .NamePadding }} {{.Short}}\n")
	builder.WriteString("{{end}}{{end}}\n")
	builder.WriteString("Flags:\n{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}\n")
	builder.WriteString("{{if .HasAvailableInheritedFlags}}\nGlobal Flags:\n{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}\n{{end}}")
	builder.WriteString("{{if .HasAvailableSubCommands}}\nUse \"{{.CommandPath}} [command] --help\" for more information about a command.\n{{end}}")

	return builder.String()
}

func loadConfigWithRecovery(showWelcome bool) (*config.AppConfig, error) {
	cfg, err := config.Load()
	if err == nil {
		return cfg, nil
	}

	switch {
	case errors.Is(err, config.ErrConfigNotFound):
		if !interactive() {
			return nil, configErrorf("aipim has not been configured yet. Run `aipim` in a terminal to set up your first profile, or `aipim create --name <n> --agent <a> --no-tui`")
		}
		cfg = config.DefaultConfig()
		if err := tui.RunOnboarding(cfg, showWelcome); err != nil {
			return nil, err
		}

		return cfg, nil
	case errors.Is(err, config.ErrConfigCorrupt):
		if !interactive() {
			return nil, configErrorf("aipim config is corrupted: %w. Move it aside or run `aipim` in a TTY to repair it", err)
		}
		reset, promptErr := tui.ConfirmConfigReset(err)
		if promptErr != nil {
			return nil, promptErr
		}
		if !reset {
			return nil, configErrorf("aipim config is corrupted: %w", err)
		}

		backupPath, backupErr := config.BackupCorruptConfig()
		if backupErr != nil {
			return nil, backupErr
		}
		logger.Infof("backed up corrupted config to %s", backupPath)

		cfg = config.DefaultConfig()
		if err := tui.RunOnboarding(cfg, true); err != nil {
			return nil, err
		}

		return cfg, nil
	default:
		return nil, configErrorf("load config: %w", err)
	}
}
