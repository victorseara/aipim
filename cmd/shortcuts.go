package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/victorseara/aipim/internal/tui"
)

var shortcutsCmd = &cobra.Command{
	Use:     "shortcuts",
	Aliases: []string{"keys"},
	Short:   "Print the TUI keyboard shortcuts",
	Long: "Print the keyboard shortcuts used by the aipim TUI.\n\n" +
		"Useful as a cheat sheet outside the TUI, or as JSON for tools and AI agents.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		shortcuts := tui.Shortcuts()

		if globalJSON {
			encodeJSON(struct {
				Shortcuts []tui.Shortcut `json:"shortcuts"`
			}{Shortcuts: shortcuts})
			return nil
		}

		if globalQuiet {
			for _, s := range shortcuts {
				fmt.Println(s.Keys)
			}
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "KEY\tACTION")
		for _, s := range shortcuts {
			fmt.Fprintf(w, "%s\t%s\n", s.Keys, s.Description)
		}
		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(shortcutsCmd)
}
