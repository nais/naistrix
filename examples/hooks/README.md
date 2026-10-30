# Pre-run lifecycle hooks

Use named application-wide hooks for authentication or other setup that must
happen **before argument choices, prompts, and validation**:

```go
app, _, err := naistrix.NewApplication("mycli", "My CLI", "v1.0.0",
    naistrix.ApplicationWithPreRunHooks(naistrix.PreRunHook{
        Name: "auth",
        RunFunc: func(ctx context.Context, out *naistrix.OutputWriter) error {
            return ensureAuthenticated(ctx, out)
        },
    }),
)
```

Hook names must be non-blank and unique, and each hook must have a `RunFunc`.
Hooks run in registration order, including across multiple
`ApplicationWithPreRunHooks` options. Configuration, global flags, inherited
sticky flags, and the selected command's flags are synchronized before any hook
runs. Hooks receive only the execution context and the application's `OutputWriter`,
not command definitions or positional arguments. Hooks perform setup; Naistrix
owns command execution, and command opt-outs remain declarative. An error is
returned unchanged and prevents later hooks, argument resolution, validation,
and command execution.

## Opting out

Skip individual hooks by name on a command or an entire command subtree:

```go
&naistrix.Command{
    Name: "auth",
    Title: "Authentication commands",
    SkipPreRunHooks: []string{"auth"},
    SubCommands: []*naistrix.Command{
        {Name: "login", Title: "Log in", RunFunc: login},
        {Name: "logout", Title: "Log out", RunFunc: logout},
    },
}
```

Skips are additive: descendants can skip more hooks, but cannot re-enable a hook
skipped by an ancestor. Unlisted hooks still run. Unknown names are ignored.
Regular aliases and top-level aliases honor the target command's skips, including
ancestor skips. A top-level alias runs the target's hooks only once.

Hooks do not run for help, version output, shell completion (including flag-value
completion), completion script generation, or parent commands that only display
subcommands.

Completion does not run pre-run hooks or initiate interactive login. Applications
should handle credential reuse and non-interactive refresh in their shared
API/authentication layer, and propagate authentication failures through completion
diagnostics. This avoids duplicating authentication checks in individual
completion callbacks.

Deprecated commands run hooks before resolving arguments and displaying the
replacement prompt. If the user accepts a replacement, that command runs its own
hooks.

The built-in `defaults` subcommands run application hooks by default. To exempt
the entire subtree from authentication, pass this option to `NewApplication`:

```go
naistrix.ApplicationWithDefaultsCommandSkipPreRunHooks("auth")
```

Other hooks still run. The option also applies if `defaults` is renamed with
`ApplicationWithDefaultsCommandName`. Multiple skip options append names, and
unknown names are ignored, just like `Command.SkipPreRunHooks`. Help always
bypasses hooks and needs no explicit opt-out.

## Running the example

```bash
# Prompts for a demo token before prompting for a member:
go run . members

# Flags are available to hooks; a supplied member still runs authentication:
go run . members Alice --token demo

# Skips the auth hook and runs the command's own login flow:
go run . auth login
go run . login

# Never starts authentication:
go run . members --help

# The example exempts defaults from auth, but other hooks would still run:
go run . defaults list
```

This example accepts any non-empty demo token and keeps it only in memory. It does
not perform real authentication or persist credentials.
