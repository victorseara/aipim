# aipim — AI Agent Profile Manager CLI

## Role & Context

You are a senior Go engineer with deep expertise in CLI tooling, TUI development using the Charm ecosystem (Bubble Tea, Lip Gloss, Huh, Bubbles), and macOS-native filesystem conventions. You build tools that feel polished, opinionated, and delightful to use — like they belong in a developer's daily workflow.

---

## What you're building

`aipim` is a **profile manager for AI coding agents**. Each profile points to a dedicated configuration directory, allowing users to maintain completely isolated agent environments (different API keys, system prompts, MCP servers, etc.) and switch between them instantly.

Think of it like browser profiles — but for AI agents in the terminal.

---

## Tech Stack (strict)

- **Language**: Go (latest stable)
- **TUI / Forms**: [Charm](https://charm.sh/) ecosystem only:
  - `github.com/charmbracelet/bubbletea` — TUI framework
  - `github.com/charmbracelet/huh` — rich interactive forms (profile creation, agent registration)
  - `github.com/charmbracelet/lipgloss` — styling and layout
  - `github.com/charmbracelet/bubbles` — list, textinput, spinner, viewport
  - `github.com/charmbracelet/log` — structured logging
- **Config**: `encoding/json` (stdlib), stored at `~/.config/aipim/`
- **CLI parsing**: `github.com/spf13/cobra`
- **Platform**: macOS only (Darwin). No Linux/Windows support needed.

---

## Project Structure

```
aipim/
├── main.go
├── cmd/
│   ├── root.go          # cobra root → opens TUI
│   ├── launch.go        # aipim launch
│   └── create.go        # aipim create
├── internal/
│   ├── config/
│   │   ├── config.go    # AppConfig struct, load/save
│   │   └── paths.go     # ~/.config/aipim path helpers
│   ├── profile/
│   │   └── profile.go   # Profile struct + CRUD
│   ├── agent/
│   │   └── agent.go     # Agent struct + built-ins + registry
│   └── tui/
│       ├── app.go       # root TUI model (list of profiles)
│       ├── create.go    # create profile form flow
│       └── styles.go    # lipgloss theme
└── go.mod
```

---

## Data Model

```go
// ~/.config/aipim/config.json
type AppConfig struct {
    DefaultAgentName string    `json:"default_agent"`
    Profiles         []Profile `json:"profiles"`
    Agents           []Agent   `json:"agents"`
}

type Profile struct {
    Name      string `json:"name"`
    Path      string `json:"path"`       // absolute path to config dir
    AgentName string `json:"agent"`      // optional; falls back to DefaultAgentName
    CreatedAt string `json:"created_at"` // RFC3339
}

type Agent struct {
    Name        string `json:"name"`
    LaunchCmd   string `json:"launch_cmd"` // e.g. "claude", "codex", "gemini", "opencode"
    IsBuiltIn   bool   `json:"built_in"`
}
```

**Built-in agents** (pre-registered, cannot be deleted):

| Name | LaunchCmd |
|------|-----------|
| Claude Code | `claude` |
| Codex | `codex` |
| Gemini | `gemini` |
| OpenCode | `opencode` |
| Copilot | `gh copilot` |

---

## Commands

### `aipim` (no args)

Opens the full **TUI** — a Bubble Tea app with:

- A list view of all profiles (name, agent, path)
- Keyboard shortcuts: `n` = new profile, `d` = delete, `enter` = launch selected, `q` = quit
- A settings panel accessible via `s` for managing agents and default agent

### `aipim launch`

- Launches the default agent with the default profile (or prompts if none configured)
- Options:
  - `-p, --profile <name>` — launch with a specific profile
  - `-m, --message <text>` — pass an initial prompt/message directly to the agent (appended after the launch command as a quoted argument)
- **What launch does**: sets the appropriate env var to the profile's `Path`, then `exec`s the agent command. This gives the agent a clean, isolated config directory.

```go
// Pseudocode for launch
cmd := exec.Command(agent.LaunchCmd, args...)
cmd.Env = append(os.Environ(), fmt.Sprintf("XDG_CONFIG_HOME=%s", profile.Path))
cmd.Stdin = os.Stdin
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
cmd.Run()
```

### `aipim create`

Runs an interactive `huh` form flow to create a new profile.

**Form steps:**

1. **Profile name** — text input, validates non-empty, no slashes, unique
2. **Config directory** — text input pre-filled with `~/.config/aipim/profiles/<name>`, user can change it. If path doesn't exist, app creates it with `os.MkdirAll`
3. **Agent selection** — `huh.Select` showing all registered agents + "Custom command…" option. If "Custom command" is selected, a text input appears for the command string.
4. **Confirm** — summary card before saving

After saving, print a success message with the launch command: `aipim launch -p <name>`

---

## First-Run Experience

On first launch (config doesn't exist yet), before opening the TUI, run a **welcome flow** using `huh`:

1. Welcome screen (styled text panel with a keypress to continue)
2. **Set default agent** — `huh.Select` from the built-in agents list + "Register custom agent…"
3. If custom: prompt for agent name + launch command
4. Save config and proceed to TUI

This should feel like a proper onboarding, not a raw prompt.

---

## TUI Design (Lipgloss)

Use a clean, dark-terminal-friendly theme:

```go
// styles.go — define these once and use everywhere
var (
    primaryColor   = lipgloss.Color("#7C3AED")   // violet
    mutedColor     = lipgloss.Color("#6B7280")
    successColor   = lipgloss.Color("#10B981")
    dangerColor    = lipgloss.Color("#EF4444")
    borderColor    = lipgloss.Color("#374151")

    titleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(primaryColor).
        MarginBottom(1)

    profileItemStyle = lipgloss.NewStyle().
        PaddingLeft(2)

    selectedItemStyle = profileItemStyle.Copy().
        Foreground(primaryColor).
        Bold(true)

    helpStyle = lipgloss.NewStyle().
        Foreground(mutedColor).
        MarginTop(1)
)
```

**List view layout:**

```
  aipim  ──────────────────────────────────────────

  ● sgws          claude code   ~/.config/aipim/profiles/sgws
    personal      gemini        ~/projects/personal-ai
    work-gpt      codex         ~/work/.ai-config

  ──────────────────────────────────────────────────
  enter launch  n new  d delete  s settings  q quit
```

---

## Agent Environment Variable Mapping

Different agents use different env vars for config directory. Handle this explicitly:

```go
var agentConfigEnvVars = map[string]string{
    "claude":     "CLAUDE_CONFIG_DIR",  // Claude Code uses this specific var
    "codex":      "XDG_CONFIG_HOME",
    "gemini":     "XDG_CONFIG_HOME",
    "opencode":   "XDG_CONFIG_HOME",
    "gh copilot": "XDG_CONFIG_HOME",
}
```

When launching, set the correct env var for the agent. Fall back to `XDG_CONFIG_HOME` for unknown/custom agents.

---

## UX Requirements & Edge Cases

Handle all of these explicitly:

1. **Profile name conflicts** — validate uniqueness on create; show inline error in `huh` form
2. **Missing agent binary** — before launching, check if the agent's command exists in `$PATH` using `exec.LookPath`. If not found: `agent "claude" not found in PATH. Install it first.`
3. **Non-existent profile on launch** — `aipim launch -p foo` where "foo" doesn't exist → clear error + list available profiles
4. **Empty config (no profiles)** — TUI shows empty state: `No profiles yet. Press n to create one.`
5. **Config directory creation** — always use `os.MkdirAll(path, 0755)`, never fail silently
6. **Config file corruption** — if JSON is invalid, show a recovery message and offer to reset
7. **Default agent not set** — if no default agent and user runs `aipim launch` without `-p`, trigger the first-run agent selection flow
8. **Profile path with `~`** — always expand `~/` to the real home directory using `os.UserHomeDir()` before storing or using paths

---

## `--help` Output Style

Use Cobra's built-in help system but customize the output with Lip Gloss colors via `cobra.SetOut`. The help should show the ASCII art logo + usage in a styled format.

---

## Deliverables

Produce the **complete, working Go source code** for this project:

1. All files in the structure above
2. `go.mod` with correct module path (`github.com/victorseara/aipim`) and all dependencies
3. A `README.md` with install instructions (`go install`) and usage examples
4. Code must compile with `go build ./...` with no errors
5. Every exported type and function must have a Go doc comment

**Write idiomatic Go.** Use `errors.New` and `fmt.Errorf` properly. No `panic` except in `init()` for truly unrecoverable states. Prefer explicit error handling over ignoring errors.

---

## Anti-patterns to avoid

- Do not use `os/exec` to shell out to anything except the final agent launch
- Do not use any non-Charm TUI library
- Do not hardcode `$HOME` — always use `os.UserHomeDir()`
- Do not store absolute paths with the literal `~` character — expand at write time
- Do not block the TUI event loop with I/O — do filesystem ops before starting the TUI or via `tea.Cmd`

---

Now write the complete implementation.