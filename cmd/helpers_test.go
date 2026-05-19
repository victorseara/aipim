package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/victorseara/aipim/internal/agent"
	"github.com/victorseara/aipim/internal/config"
	"github.com/victorseara/aipim/internal/profile"
)

// runAipim executes the aipim CLI in-process and returns the captured stdout,
// stderr, and exit code as the real binary would emit them.
//
// It isolates state per call:
//   - resets all command-level globals and Cobra's "Changed" flag markers
//   - redirects os.Stdout/os.Stderr to in-memory pipes
//   - runs main.go's error-handling pipeline (ExitCodeFor + emitError)
func runAipim(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	resetState(t)

	oldStdout, oldStderr := os.Stdout, os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stdout: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stderr: %v", err)
	}
	os.Stdout = wOut
	os.Stderr = wErr

	rootCmd.SetArgs(args)
	execErr := rootCmd.Execute()
	code = ExitCodeFor(execErr)
	if execErr != nil {
		emitError(execErr, code)
	}

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var outBuf, errBuf bytes.Buffer
	_, _ = io.Copy(&outBuf, rOut)
	_, _ = io.Copy(&errBuf, rErr)
	return outBuf.String(), errBuf.String(), code
}

// resetState resets every package-level CLI variable and Cobra flag marker so each
// test starts from a clean slate. This is the price of having a singleton rootCmd.
func resetState(t *testing.T) {
	t.Helper()
	config.SetConfigDirOverride("")

	globalJSON = false
	globalQuiet = false
	globalNoTUI = false
	globalConfDir = ""
	rootMessage = ""

	listWithDescription = false
	deleteYes = false
	deleteKeepFiles = false

	createName = ""
	createAlias = ""
	createAgent = ""
	createPath = ""
	createDescription = ""
	createSetAsDefault = false

	editAlias = ""
	editAgent = ""
	editPath = ""
	editDescription = ""
	editClearAlias = false
	editAliasSet = false
	editAgentSet = false
	editPathSet = false
	editDescriptionSet = false

	agentAddName = ""
	agentAddCmd = ""

	launchProfileName = ""
	launchMessage = ""

	resetCobraFlagChanged(rootCmd)
}

// resetCobraFlagChanged recursively unsets the Changed bit on every flag so a
// previous test's `cmd.Flags().Changed("...")` does not leak into the next one.
func resetCobraFlagChanged(c *cobra.Command) {
	reset := func(f *pflag.Flag) { f.Changed = false }
	c.Flags().VisitAll(reset)
	c.PersistentFlags().VisitAll(reset)
	for _, child := range c.Commands() {
		resetCobraFlagChanged(child)
	}
}

// withTempConfig points AIPIM_CONFIG_HOME at a fresh temp directory for the
// duration of the test and seeds the given AppConfig if non-nil.
func withTempConfig(t *testing.T, seed *config.AppConfig) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(config.EnvConfigHome, dir)
	if seed != nil {
		if err := seed.Save(); err != nil {
			t.Fatalf("seed config: %v", err)
		}
	}
	return dir
}

// newProfile returns a Profile with the boring fields filled in.
func newProfile(name, alias, agentName, description string) profile.Profile {
	return profile.Profile{
		Name:        name,
		Alias:       alias,
		AgentName:   agentName,
		Description: description,
		Path:        "",
		CreatedAt:   time.Now().Format(time.RFC3339),
	}
}

// seedDefaults returns a config with two profiles assigned to Claude Code/Codex.
func seedDefaults() *config.AppConfig {
	return &config.AppConfig{
		DefaultAgentName: "Claude Code",
		Agents:           agent.BuiltIns(),
		Profiles: []profile.Profile{
			newProfile("sgws", "s", "Claude Code", "SGWS client work. Snowflake MCP."),
			newProfile("personal", "", "Codex", "Personal weekend projects."),
		},
	}
}
