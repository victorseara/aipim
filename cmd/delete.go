package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/victorseara/aipim/internal/config"
	"github.com/victorseara/aipim/internal/profile"
)

var (
	deleteYes       bool
	deleteKeepFiles bool

	deleteCmd = &cobra.Command{
		Use:     "delete <name|alias>",
		Aliases: []string{"rm"},
		Short:   "Delete a profile",
		Long: "Delete a profile from the aipim config.\n\n" +
			"By default the profile's config directory is also removed. Pass --keep-files to keep it on disk.\n" +
			"Pass --yes to skip the confirmation prompt (required for scripts and CI).",
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

			if !deleteYes {
				if !interactive() {
					return usageErrorf("delete requires confirmation. Re-run with --yes")
				}
				var confirmed bool
				form := huh.NewForm(
					huh.NewGroup(
						huh.NewConfirm().
							Title("Delete profile").
							Description(fmt.Sprintf("Delete profile %q? This cannot be undone.", found.Name)).
							Affirmative("Delete").
							Negative("Cancel").
							Value(&confirmed),
					),
				)
				if err := form.Run(); err != nil {
					return err
				}
				if !confirmed {
					return cancelledError
				}
			}

			updated, err := profile.DeleteByName(cfg.Profiles, found.Name)
			if err != nil {
				return configErrorf("%w", err)
			}
			cfg.Profiles = updated
			if err := cfg.Save(); err != nil {
				return configErrorf("save config: %w", err)
			}

			if !deleteKeepFiles {
				profilePath := strings.TrimSpace(found.Path)
				if profilePath != "" {
					if err := os.RemoveAll(profilePath); err != nil {
						return configErrorf("remove profile directory %q: %w (config updated but files remain)", profilePath, err)
					}
				}
			}

			if globalJSON {
				printDeleted(found.Name)
			} else if !globalQuiet {
				fmt.Fprintf(os.Stdout, "Deleted profile %q.\n", found.Name)
			}
			return nil
		},
	}
)

func init() {
	deleteCmd.Flags().BoolVarP(&deleteYes, "yes", "y", false, "Skip the confirmation prompt")
	deleteCmd.Flags().BoolVar(&deleteKeepFiles, "keep-files", false, "Keep the profile's config directory on disk")
	rootCmd.AddCommand(deleteCmd)
}

func printDeleted(name string) {
	payload := struct {
		Deleted string `json:"deleted"`
	}{Deleted: name}
	encodeJSON(payload)
}
