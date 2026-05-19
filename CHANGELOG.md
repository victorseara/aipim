# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `aipim list` / `aipim ls` — table, `--json`, and `--quiet` output modes.
- `aipim get <name>` (alias `show`) — full single-profile detail with `--json`.
- `aipim edit <name>` — CLI patch flags (`--alias`, `--clear-alias`, `--agent`, `--path`, `--description`) plus TUI fallback.
- `aipim delete <name>` (alias `rm`) — non-interactive with `--yes`; removes the profile directory unless `--keep-files`.
- `aipim agent` — `list` / `add` / `rm` / `default` subcommands.
- Non-interactive `aipim create` via `--name`, `--alias`, `--agent`, `--path`, `--description`, `--set-default`.
- `Description` field on profiles — surfaced in `list` / `get` for AI-agent profile selection from natural-language prompts.
- `--version` flag with ldflags-injected version, commit, and build date.
- Global flags: `--json`, `--quiet`, `--no-tui`, `--config-dir`.
- `AIPIM_CONFIG_HOME` and `XDG_CONFIG_HOME` resolution for the config directory.
- Shell completions via Cobra: `aipim completion bash|zsh|fish|powershell` with dynamic profile-name completion.
- `?` opens a full-screen help overlay in the TUI; inline description preview under the highlighted profile.
- Deterministic exit codes (0/1/2/3/4/5) and a structured `ExitError` type.
- JSON error envelope (`{"error":"…","code":N}`) on `--json`.
- Makefile, `.goreleaser.yaml`, LICENSE, and CHANGELOG.

### Changed
- Module path renamed from `github.com/user/aipim` to `github.com/victorseara/aipim` so `go install` works.
- Error messages now include remediation hints (run `aipim list`, install the missing binary, etc.).
- Reserved profile-name list expanded to cover all new top-level subcommands.
- TUI legend now collapses gracefully on narrow terminals; full keymap available via `?`.

## [0.1.0] - Initial release seed

- Profile-managed launches for Claude Code, Codex, Gemini, OpenCode, and Copilot.
- Interactive TUI for browsing, creating, editing, and deleting profiles.
- JSON config at `~/.config/aipim/config.json`.
