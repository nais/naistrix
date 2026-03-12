# AGENTS.md

## Overview
Naistrix is an opinionated Go library for building CLI applications, wrapping [Cobra](https://github.com/spf13/cobra) and providing conventions for commands, flags, arguments, output, and developer workflow. This document summarizes essential project knowledge for AI coding agents to be productive in this codebase.

## Architecture & Key Components
- **Core Types**: The main abstractions are `Application`, `Command`, `Argument`, `OutputWriter`, and flag structs. See `application.go`, `command.go`, `arguments.go`, and `output/`.
- **Command Structure**: Commands are defined via the `Command` struct, supporting subcommands, aliases, top-level aliases, groups, arguments, flags, sticky flags, and deprecation. See `command.go` and `examples/`.
- **Flags & Arguments**: Flags are defined as struct fields (with tags for name, short, usage, etc.), and arguments are positional, defined in the `Args` field of `Command`. Repeatable arguments must be last. See `examples/flags/README.md` and `examples/arguments/README.md`.
- **Output**: Output is handled via the `OutputWriter` abstraction, supporting verbosity, pretty output, tables, JSON, and YAML. See `output/` and `examples/`.
- **Deprecation**: Commands can be deprecated with static or dynamic replacements, or without replacement. See `examples/deprecated/README.md`.
- **Color & Styling**: Use the `internal/color` package for colorized output with custom tags (e.g., `<info>`, `<warn>`, `<error>`).

## Developer Workflow
- **Dependency & Task Management**: Uses [mise](https://mise.jdx.dev/) for tool and task management. See `mise/config.toml` and `mise/tasks/`.
- **Common Tasks**:
  - `mise run build`: Build all Go code (`mise/tasks/build.sh`)
  - `mise run fmt`: Format code with gofumpt (`mise/tasks/fmt.sh`)
  - `mise run test`: Run tests with coverage and race detection (`mise/tasks/test.sh`)
  - `mise run check:golangci-lint`, `check:govet`, `check:govulncheck`, etc. for static analysis and security
- **Conventional Commits**: All commits must follow [conventional commit](https://www.conventionalcommits.org/en/v1.0.0/) style. A commit-msg Git hook is provided (`script/conventional-commit-hook.sh`).
- **Examples**: The `examples/` directory contains self-contained usage examples for all major features (flags, arguments, aliases, output, deprecation, etc.). When adding new features, include an example here.

## Project Conventions & Patterns
- **No Spaces in Command Names**: Command names must not contain spaces. Use hyphens for multi-word commands (e.g., `my-command`).
- **Mutual Exclusivity**: `RunFunc` and `SubCommands` are mutually exclusive in a `Command`.
- **Repeatable Arguments**: Only the last argument can be repeatable.
- **Aliases**: Aliases and top-level aliases must be unique at their respective levels.
- **Deprecation**: Parent commands with subcommands cannot be deprecated; deprecate subcommands individually.
- **Flag Struct Tags**: Use struct tags (`name`, `short`, `usage`) to configure flags.
- **Output**: Prefer using `OutputWriter` for all user-facing output, supporting verbosity and pretty formatting.
- **Colorization**: Use `<info>`, `<warn>`, `<error>` tags for colorized output via `internal/color`.

## Integration Points
- **Cobra**: All command logic ultimately wraps Cobra commands.
- **Viper**: Used for configuration management.
- **pterm**: Used for styled terminal output.
- **gofumpt**, **golangci-lint**, **govulncheck**: Used for formatting, linting, and vulnerability checks.

## References
- Main entrypoints: `application.go`, `command.go`, `arguments.go`, `output/`
- Developer workflow: `README.md`, `mise/config.toml`, `mise/tasks/`, `script/conventional-commit-hook.sh`
- Usage patterns: `examples/`
- Colorization: `internal/color/`

---
For more, see [README.md] and the `examples/` directory for practical usage patterns.

