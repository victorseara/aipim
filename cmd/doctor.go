package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/google/shlex"
	"github.com/spf13/cobra"

	"github.com/victorseara/aipim/internal/agent"
	"github.com/victorseara/aipim/internal/config"
	"github.com/victorseara/aipim/internal/profile"
)

// DoctorReport is the agent-readable health snapshot for the aipim install.
// Stable JSON shape — see README "Agent API reference".
type DoctorReport struct {
	OK       bool                  `json:"ok"`
	Errors   []string              `json:"errors,omitempty"`
	Warnings []string              `json:"warnings,omitempty"`
	Profiles []ProfileDiagnostic   `json:"profiles"`
	Agents   []AgentDiagnostic     `json:"agents"`
}

type ProfileDiagnostic struct {
	Name         string   `json:"name"`
	OK           bool     `json:"ok"`
	Agent        string   `json:"agent"`
	AgentExists  bool     `json:"agent_registered"`
	BinaryFound  bool     `json:"binary_found"`
	Path         string   `json:"path"`
	PathStatus   string   `json:"path_status"` // "ok", "missing", "not-writable", "agent-managed"
	HasDescription bool   `json:"has_description"`
	Errors       []string `json:"errors,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

type AgentDiagnostic struct {
	Name        string `json:"name"`
	LaunchCmd   string `json:"launch_cmd"`
	BinaryFound bool   `json:"binary_found"`
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Validate the aipim configuration",
	Long: "Validate the aipim configuration and the local environment.\n\n" +
		"Checks each profile's assigned agent is registered, its binary is on PATH, the profile directory exists and is writable, and the description is populated.\n\n" +
		"Exits with code 0 if everything is healthy; code 1 if any profile has errors. Warnings (e.g. missing description) do not affect the exit code.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return loadErrToExit(err)
		}

		report := buildDoctorReport(cfg)

		if globalJSON {
			encodeJSON(report)
			if !report.OK {
				return silentExit(ExitGeneric)
			}
			return nil
		}

		printDoctorHuman(report)
		if !report.OK {
			return silentExit(ExitGeneric)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func buildDoctorReport(cfg *config.AppConfig) DoctorReport {
	report := DoctorReport{OK: true}

	if strings.TrimSpace(cfg.DefaultAgentName) == "" {
		report.Warnings = append(report.Warnings, "no default agent set; profiles without an explicit agent cannot be launched")
	}

	for _, a := range cfg.Agents {
		report.Agents = append(report.Agents, AgentDiagnostic{
			Name:        a.Name,
			LaunchCmd:   a.LaunchCmd,
			BinaryFound: binaryInPath(a.LaunchCmd),
		})
	}

	for _, p := range cfg.Profiles {
		diag := diagnoseProfile(cfg, p)
		report.Profiles = append(report.Profiles, diag)
		if !diag.OK {
			report.OK = false
			report.Errors = append(report.Errors, fmt.Sprintf("profile %q: %s", diag.Name, strings.Join(diag.Errors, "; ")))
		}
	}

	if len(cfg.Profiles) == 0 {
		report.Warnings = append(report.Warnings, "no profiles configured; run `aipim create` to add one")
	}

	return report
}

func diagnoseProfile(cfg *config.AppConfig, p profile.Profile) ProfileDiagnostic {
	d := ProfileDiagnostic{
		Name:           p.Name,
		OK:             true,
		Path:           p.Path,
		HasDescription: strings.TrimSpace(p.Description) != "",
	}

	agentName := strings.TrimSpace(p.AgentName)
	if agentName == "" {
		agentName = strings.TrimSpace(cfg.DefaultAgentName)
	}
	d.Agent = agentName

	if agentName == "" {
		d.OK = false
		d.Errors = append(d.Errors, "no agent assigned and no default agent set")
	} else if a, ok := agent.FindByName(cfg.Agents, agentName); !ok {
		d.OK = false
		d.Errors = append(d.Errors, fmt.Sprintf("agent %q is not registered", agentName))
	} else {
		d.AgentExists = true
		d.BinaryFound = binaryInPath(a.LaunchCmd)
		if !d.BinaryFound {
			d.OK = false
			d.Errors = append(d.Errors, fmt.Sprintf("agent binary %q not found in PATH", firstToken(a.LaunchCmd)))
		}
	}

	if strings.TrimSpace(p.Path) == "" {
		d.PathStatus = "agent-managed"
	} else {
		expanded, err := config.ExpandPath(p.Path)
		if err != nil {
			d.OK = false
			d.PathStatus = "invalid"
			d.Errors = append(d.Errors, fmt.Sprintf("invalid path %q: %v", p.Path, err))
		} else if info, err := os.Stat(expanded); err != nil {
			if os.IsNotExist(err) {
				d.PathStatus = "missing"
				d.Warnings = append(d.Warnings, fmt.Sprintf("path %q does not exist (will be created at launch)", expanded))
			} else {
				d.OK = false
				d.PathStatus = "error"
				d.Errors = append(d.Errors, fmt.Sprintf("stat %q: %v", expanded, err))
			}
		} else if !info.IsDir() {
			d.OK = false
			d.PathStatus = "not-a-directory"
			d.Errors = append(d.Errors, fmt.Sprintf("path %q is not a directory", expanded))
		} else if !isWritable(expanded) {
			d.OK = false
			d.PathStatus = "not-writable"
			d.Errors = append(d.Errors, fmt.Sprintf("path %q is not writable", expanded))
		} else {
			d.PathStatus = "ok"
		}
	}

	if !d.HasDescription {
		d.Warnings = append(d.Warnings, "no description; AI agents have no signal to pick this profile")
	}

	return d
}

// binaryInPath looks up the first token of a shell-style launch command and
// reports whether it resolves on PATH.
func binaryInPath(launchCmd string) bool {
	parts, err := shlex.Split(strings.TrimSpace(launchCmd))
	if err != nil || len(parts) == 0 {
		return false
	}
	_, err = exec.LookPath(parts[0])
	return err == nil
}

func firstToken(launchCmd string) string {
	parts, err := shlex.Split(strings.TrimSpace(launchCmd))
	if err != nil || len(parts) == 0 {
		return launchCmd
	}
	return parts[0]
}

func isWritable(path string) bool {
	probe, err := os.CreateTemp(path, ".aipim-doctor-*")
	if err != nil {
		return false
	}
	_ = probe.Close()
	_ = os.Remove(probe.Name())
	return true
}

func printDoctorHuman(report DoctorReport) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROFILE\tAGENT\tBINARY\tPATH\tDESCRIPTION\tSTATUS")
	for _, d := range report.Profiles {
		status := "ok"
		if !d.OK {
			status = "ERROR"
		} else if len(d.Warnings) > 0 {
			status = "warn"
		}
		binary := "not-found"
		if d.BinaryFound {
			binary = "ok"
		} else if !d.AgentExists {
			binary = "agent-unregistered"
		}
		desc := "missing"
		if d.HasDescription {
			desc = "set"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", d.Name, d.Agent, binary, d.PathStatus, desc, status)
	}
	_ = w.Flush()

	for _, d := range report.Profiles {
		for _, e := range d.Errors {
			fmt.Fprintf(os.Stdout, "\n  ✗ %s: %s\n", d.Name, e)
		}
		for _, warn := range d.Warnings {
			fmt.Fprintf(os.Stdout, "  ! %s: %s\n", d.Name, warn)
		}
	}
	for _, w := range report.Warnings {
		fmt.Fprintf(os.Stdout, "\n  ! %s\n", w)
	}
	for _, e := range report.Errors {
		fmt.Fprintf(os.Stdout, "\n  ✗ %s\n", e)
	}
	if report.OK && len(report.Warnings) == 0 {
		fmt.Println()
		fmt.Println("All profiles healthy.")
	}
}
