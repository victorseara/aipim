package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

// Context is the snapshot of selection signals an agent uses to pick a profile.
// Stable JSON shape — see README "Agent API reference".
type Context struct {
	CWD string         `json:"cwd"`
	Git GitContext     `json:"git"`
	GH  GHContext      `json:"gh"`
}

type GitContext struct {
	Available bool   `json:"available"`            // false if cwd is not inside a git repo
	RemoteURL string `json:"remote_url,omitempty"` // origin remote
	RemoteOrg string `json:"remote_org,omitempty"` // parsed owner from the remote URL
	UserEmail string `json:"user_email,omitempty"`
	UserName  string `json:"user_name,omitempty"`
}

type GHContext struct {
	Available      bool     `json:"available"` // false if `gh` is not on PATH
	ActiveAccount  string   `json:"active_account,omitempty"`
	OtherAccounts  []string `json:"other_accounts,omitempty"`
}

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Print the current selection signals (cwd, git, gh)",
	Long: "Print the current selection signals — cwd, git remote/email, gh active account — that an AI agent uses to pick a profile.\n\n" +
		"With --json, emits a stable structured snapshot suitable for piping into the profile-selection logic.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := gatherContext()

		if globalJSON {
			encodeJSON(ctx)
			return nil
		}

		fmt.Printf("cwd: %s\n", ctx.CWD)
		fmt.Println("git:")
		if !ctx.Git.Available {
			fmt.Println("  (not a git repository)")
		} else {
			if ctx.Git.RemoteURL != "" {
				fmt.Printf("  remote:     %s\n", ctx.Git.RemoteURL)
				if ctx.Git.RemoteOrg != "" {
					fmt.Printf("  remote_org: %s\n", ctx.Git.RemoteOrg)
				}
			}
			if ctx.Git.UserEmail != "" {
				fmt.Printf("  user_email: %s\n", ctx.Git.UserEmail)
			}
			if ctx.Git.UserName != "" {
				fmt.Printf("  user_name:  %s\n", ctx.Git.UserName)
			}
		}
		fmt.Println("gh:")
		if !ctx.GH.Available {
			fmt.Println("  (gh CLI not on PATH)")
		} else {
			if ctx.GH.ActiveAccount != "" {
				fmt.Printf("  active:     %s\n", ctx.GH.ActiveAccount)
			}
			if len(ctx.GH.OtherAccounts) > 0 {
				fmt.Printf("  available:  %s\n", strings.Join(ctx.GH.OtherAccounts, ", "))
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(contextCmd)
}

func gatherContext() Context {
	ctx := Context{}
	if cwd, err := os.Getwd(); err == nil {
		ctx.CWD = cwd
	}
	ctx.Git = gatherGit()
	ctx.GH = gatherGH()
	return ctx
}

func gatherGit() GitContext {
	g := GitContext{}
	if _, err := exec.LookPath("git"); err != nil {
		return g
	}

	// Confirm we're inside a working tree before running other queries.
	if out, err := runQuiet("git", "rev-parse", "--is-inside-work-tree"); err != nil || strings.TrimSpace(out) != "true" {
		// Still try config queries — they read global config too.
		if email, err := runQuiet("git", "config", "--global", "--get", "user.email"); err == nil {
			g.UserEmail = strings.TrimSpace(email)
		}
		if name, err := runQuiet("git", "config", "--global", "--get", "user.name"); err == nil {
			g.UserName = strings.TrimSpace(name)
		}
		return g
	}

	g.Available = true

	if out, err := runQuiet("git", "remote", "get-url", "origin"); err == nil {
		g.RemoteURL = strings.TrimSpace(out)
		g.RemoteOrg = parseRemoteOrg(g.RemoteURL)
	}
	if out, err := runQuiet("git", "config", "--get", "user.email"); err == nil {
		g.UserEmail = strings.TrimSpace(out)
	}
	if out, err := runQuiet("git", "config", "--get", "user.name"); err == nil {
		g.UserName = strings.TrimSpace(out)
	}
	return g
}

// parseRemoteOrg extracts the owner segment from common git remote URL shapes:
//   - git@github.com:owner/repo.git
//   - https://github.com/owner/repo.git
//   - ssh://git@github.com/owner/repo
// Returns an empty string if no owner can be confidently extracted.
var remoteOrgRe = regexp.MustCompile(`(?:[:/])([^/:]+)/[^/]+?(?:\.git)?/?$`)

func parseRemoteOrg(remote string) string {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return ""
	}
	m := remoteOrgRe.FindStringSubmatch(remote)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func gatherGH() GHContext {
	g := GHContext{}
	if _, err := exec.LookPath("gh"); err != nil {
		return g
	}
	g.Available = true

	if out, err := runQuiet("gh", "auth", "status", "--active"); err == nil {
		g.ActiveAccount = extractGHActiveLogin(out)
	}

	if out, err := runQuiet("gh", "auth", "status"); err == nil {
		for _, login := range extractAllGHLogins(out) {
			if login == g.ActiveAccount {
				continue
			}
			g.OtherAccounts = append(g.OtherAccounts, login)
		}
	}
	return g
}

// extractGHActiveLogin parses `gh auth status --active` text output to find the
// active account login. The output format is stable enough — `Logged in to
// github.com account <login>` — but we look for it line-by-line so we don't
// break on minor wording changes.
var ghLoginRe = regexp.MustCompile(`account ([^\s]+)`)

func extractGHActiveLogin(text string) string {
	for _, line := range strings.Split(text, "\n") {
		m := ghLoginRe.FindStringSubmatch(line)
		if len(m) == 2 {
			return m[1]
		}
	}
	return ""
}

func extractAllGHLogins(text string) []string {
	seen := map[string]bool{}
	var out []string
	for _, line := range strings.Split(text, "\n") {
		m := ghLoginRe.FindStringSubmatch(line)
		if len(m) == 2 && !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	return out
}

// runQuiet runs a command and returns stdout. Stderr is discarded.
func runQuiet(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Stderr = nil
	out, err := cmd.Output()
	return string(out), err
}
