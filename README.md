# Naistrix

[![Go Reference](https://pkg.go.dev/badge/github.com/nais/naistrix.svg)](https://pkg.go.dev/github.com/nais/naistrix)

Naistrix is an opinionated wrapper around the [Cobra library](https://github.com/spf13/cobra), used when building CLI applications in Go.

## Installation

Use `go get` to fetch the latest version of Naistrix:

```bash
go get github.com/nais/naistrix@latest
```

## Non-interactive execution

Use `--non-interactive` to disable framework and context-aware prompts even when a
terminal is available. The invocation policy is established before pre-run hooks
and argument resolution. Missing required arguments fail, and confirmations return
`input.ErrNotInteractive`; the flag never implies yes, consent or permission.
Fully specified commands, configured defaults and explicit single-option
auto-selection continue to work without prompting.

Commands and hooks must pass their context to `input.InputContext`,
`ConfirmContext`, `SelectContext` and `SecretContext`, or check
`input.IsInteractiveContext(ctx)` before starting their own interactive flow.
Embedders can also use `input.WithNonInteractive(ctx)` with `RunWithContext`.
The policy is context-scoped, not a process-global toggle.

Existing helpers without a context retain their terminal-only behavior and cannot
honor this invocation policy; migrate those calls to the context-aware variants.
External prompt libraries must likewise be gated by the application. Cancellation
is checked before prompting, not while a prompt is running. The
[input](examples/input/) and [hook](examples/hooks/) examples demonstrate the APIs.

## Usage

Please have a look at the [examples](examples/) directory for usage examples.

### Application lifecycle

An `Application` is single-use: create it, register commands and flags, then call
`Run` once. A failed run still consumes the application; create a new application
to retry or execute another invocation, including in tests. Subsequent calls to
`Run` return `naistrix.ErrApplicationAlreadyRun` without executing commands.

Aliases and deprecated-command replacements are handled within the original run
and do not require another call to `Run`. Applications are not safe for concurrent
use.

## Local development

### Clone the repository:

```bash
git clone git@github.com:nais/naistrix.git
cd naistrix
```

### Install tools

Naistrix uses [mise](https://mise.jdx.dev/) to handle dependencies and tasks. After installing mise run the following command to install tools:

```bash
mise install
```

### Make changes

First, create a new branch for your changes:

```bash
git checkout -b my-feature-branch
```

Make the necessary changes to the codebase. When committing your changes, make sure to follow the [conventional commit specification](https://www.conventionalcommits.org/en/v1.0.0/). This helps maintain a clean and understandable project history along with a nice automated Changelog.

The repository contains a [`commit-msg`](script/semantic-commit-hook.sh) git hook that might be helpful. To enable this you can do the following from the project root:

```bash
ln -s ../../script/semantic-commit-hook.sh .git/hooks/commit-msg
```

There are also several mise tasks that you can use to validate your changes. See all available tasks by running:

```bash
mise run
```

### Create pull request

Once you have made your changes and committed them, push your branch and make a pull request against the `main` branch.

Use a Conventional Commit title for the pull request as well, for example `feat: document configuration keys`. When squash-merging, keep the final squash commit title conventional: release generation reads that title, not the individual commit messages included in its body.

## License

MIT, see [LICENSE.txt](LICENSE.txt) for details.
