# aipim

`aipim` is a profile manager for AI coding agents. Each profile points to an isolated config directory, so you can switch between Claude Code, Codex, Gemini, OpenCode, Copilot, or any custom CLI without their configs ever mixing.

It works two ways:

- **As a TUI** for humans — `aipim` opens an interactive list, you press `enter`, the agent launches with the right env.
- **As a scriptable CLI** for AI agents and automation — every action is reachable through flagged subcommands with JSON output, deterministic exit codes, shell completion, and an env override for sandboxed runs.

## Install

**Pre-built binary (no Go required):**

```bash
# Linux/macOS, amd64 or arm64
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
VERSION=$(curl -s https://api.github.com/repos/victorseara/aipim/releases/latest | grep tag_name | cut -d '"' -f 4)
curl -L "https://github.com/victorseara/aipim/releases/download/${VERSION}/aipim_${VERSION#v}_${OS}_${ARCH}.tar.gz" \
  | tar -xz -C /tmp aipim
sudo mv /tmp/aipim /usr/local/bin/
aipim --version
```

Or pick the right archive manually from the [releases page](https://github.com/victorseara/aipim/releases).

**With Go:**

```bash
go install github.com/victorseara/aipim@latest
```

**From source:**

```bash
git clone https://github.com/victorseara/aipim && cd aipim
make install   # honours ldflags for Version/Commit/Date
```

## Quickstart

Open the TUI and create your first profile:

```bash
aipim
```

Launch a profile directly:

```bash
aipim work                   # by name
aipim w                      # by alias
aipim launch -p work -m "summarize the repo"
aipim launch -p work -- -p "verbatim agent args after --"
```

## CLI reference

| Command | Purpose | Key flags |
| --- | --- | --- |
| `aipim` | Open TUI, or launch `aipim <profile>` directly | `-m, --message` |
| `aipim list` (alias `ls`) | List profiles | `--json`, `--quiet`, `--with-description` |
| `aipim get <name>` (alias `show`) | Show one profile in full | `--json` |
| `aipim create` | Create a profile | `--name`, `--alias`, `--agent`, `--path`, `--description`, `--set-default` |
| `aipim edit <name>` | Patch a profile | `--alias`, `--clear-alias`, `--agent`, `--path`, `--description` |
| `aipim delete <name>` (alias `rm`) | Delete a profile | `-y, --yes`, `--keep-files` |
| `aipim launch [-p name]` | Launch an agent with a profile | `-p, --profile`, `-m, --message` |
| `aipim agent list` | List registered agents | `--json`, `--quiet` |
| `aipim agent add` | Register a custom agent | `--name`, `--cmd` |
| `aipim agent rm <name>` | Remove a custom agent | — |
| `aipim agent default <name>` | Set the default agent | — |
| `aipim completion <shell>` | Generate shell completions | `bash`, `zsh`, `fish`, `powershell` |
| `aipim shortcuts` (alias `keys`) | Print the TUI keyboard shortcuts | `--json`, `--quiet` |
| `aipim --version` | Print version, commit, build date | — |

### Global flags

| Flag | Effect |
| --- | --- |
| `--config-dir <path>` | Override the aipim config directory for this invocation. Equivalent to `AIPIM_CONFIG_HOME`. |
| `--json` | Emit JSON output (where supported) and JSON error envelopes. Implies `--no-tui`. |
| `--quiet` | Suppress non-essential stdout output. |
| `--no-tui` | Refuse to open the TUI; fail with an exit code when interaction would be required. |

## AI-agent recipes

`aipim` is designed to be safely orchestrated by AI agents. Every read command supports `--json`, every write command supports flag-driven non-interactive mode, and errors come back as a single JSON line with a numeric exit code when `--json` is set.

**Pick the right profile from a user's prompt:**

```bash
# Enumerate profiles with their selection hints
aipim list --json | jq '.profiles[] | {name, alias, description}'

# Match by keyword in the description
aipim list --json \
  | jq -r '.profiles[] | select(.description | test("snowflake"; "i")) | .name'

# Launch the chosen profile (replaces the current process via execve)
aipim launch -p sgws -- -p "$USER_PROMPT"
```

**Create profiles non-interactively:**

```bash
aipim create --no-tui \
  --name sgws \
  --alias s \
  --agent "Claude Code" \
  --description "SGWS client work. Snowflake MCP + design-system Figma file pre-loaded."
```

**Read a long description from stdin:**

```bash
cat my-context.md | aipim edit sgws --description -
```

**Use an ephemeral config directory (CI / sandbox):**

```bash
export AIPIM_CONFIG_HOME=$(mktemp -d)
aipim create --no-tui --name ci --agent "Claude Code" --description "CI runs"
aipim list --quiet
```

## Built-in agents

| Name | Launch command | Config env var |
| --- | --- | --- |
| Claude Code | `claude` | `CLAUDE_CONFIG_DIR` |
| Codex | `codex` | `XDG_CONFIG_HOME` |
| Gemini | `gemini` | `XDG_CONFIG_HOME` |
| OpenCode | `opencode` | `XDG_CONFIG_HOME` |
| Copilot | `gh copilot` | `XDG_CONFIG_HOME` |

Register your own with `aipim agent add --name "My Agent" --cmd "/usr/local/bin/my-agent --fast"`.

## Configuration

`aipim` stores its state under a configurable directory.

**Resolution order** (highest precedence first):

1. `--config-dir <path>` flag on the current invocation.
2. `$AIPIM_CONFIG_HOME` environment variable.
3. `$XDG_CONFIG_HOME/aipim`.
4. `~/.config/aipim`.

Files inside:

- `config.json` — profiles, agents, default agent (single source of truth).
- `profiles/<name>/` — each profile's isolated config directory (unless you've supplied your own path).
- `config.corrupt-<timestamp>.json` — backup written automatically when the TUI is asked to recover from a corrupt config.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success |
| `1` | Generic / unexpected error |
| `2` | Usage error — bad flag, unknown profile, conflicting input |
| `3` | Config error — config missing, corrupt, or invalid for the requested operation |
| `4` | Agent binary not found in `PATH` |
| `5` | User cancelled (TUI dismissed, confirmation rejected) |

When `--json` is set, errors are emitted as `{"error":"…","code":N}` on stdout and the process exits with `N`. Otherwise errors print to stderr.

## Troubleshooting

**`agent binary "claude" not found in PATH`**
Install the agent (`npm i -g @anthropic-ai/claude-code`, `brew install gemini-cli`, etc.) or register a different binary with `aipim agent add`.

**`aipim list` says "config not found" inside a Docker container / CI**
Set `AIPIM_CONFIG_HOME` to a writable path, or bind-mount your host's `~/.config/aipim` into the container.

**`profile "X" does not exist`**
Run `aipim list` to see the current set. Aliases and names are matched case-insensitively, but the cobra reserved words (`list`, `get`, `edit`, `delete`, `agent`, `version`, `completion`, `launch`, `create`, `help`) cannot be used as profile names.

**Corrupt `config.json`**
Re-run `aipim` interactively. It will offer to back the file up as `config.corrupt-<timestamp>.json` and rebuild from onboarding.

## Platform support

`aipim` targets Linux and macOS. The launch path uses `syscall.Exec` to replace the current process with the agent, which is not available on Windows. WSL works.

## Contributing

PRs welcome. Please open an issue first for anything non-trivial.

- `make test` — run unit tests.
- `make build` — produce `./dist/aipim` with version metadata injected.
- `make completions` — write shell completion scripts to `dist/completions/`.

## License

MIT — see [LICENSE](./LICENSE).
