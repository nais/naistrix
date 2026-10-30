# Pre-run lifecycle hooks

Use named application-wide or command-scoped hooks for authentication or other setup that must
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

Hook names must be non-blank and unique on each command path, and each hook must have a `RunFunc`.
Application hooks run first in registration order, including across multiple
`ApplicationWithPreRunHooks` options. Command hooks then run from ancestor to descendant,
in registration order within each command. Configuration, global flags, inherited
sticky flags, and the selected command's flags are synchronized before any hook
runs. Hooks receive only the execution context and the application's `OutputWriter`,
not command definitions or positional arguments. Hooks perform setup; Naistrix
owns command execution, and command opt-outs remain declarative. An error is
returned unchanged and prevents later hooks, argument resolution, validation,
and command execution.

## Command-scoped hooks

Register setup only on the commands that need it with `Command.PreRunHooks`. Hooks
on a parent are inherited by all its descendants, so authentication can be scoped
to a command subtree rather than registered globally and skipped elsewhere:

```go
// In this example, "auth" is not also registered application-wide.
app.AddCommand(&naistrix.Command{
    Name: "projects",
    Title: "Project commands",
    PreRunHooks: []naistrix.PreRunHook{
        {Name: "auth", RunFunc: ensureAuthenticated},
    },
    SubCommands: []*naistrix.Command{
        {Name: "list", Title: "List projects", RunFunc: listProjects},
        {
            Name: "public", Title: "Public information",
            SkipPreRunHooks: []string{"auth"},
            RunFunc: publicInformation,
        },
    },
})
```

Hook definitions are validated during registration. `AddCommand` rejects empty or
blank names, missing functions, and duplicate names along a path, including names
already registered application-wide or on an ancestor. Skipping a name does not
make a duplicate definition valid. The same hook name can be reused on sibling
commands or unrelated branches, where the hooks cannot run together.

Command hooks use the same execution context, output writer, synchronization,
error handling, and help/completion exclusions as application hooks. They run
once per execution, including when a command is invoked through a top-level alias.
Parent hooks do not run when the parent only displays its list of subcommands.

## Opting out

Skip individual application or command hooks by name on a command or an entire command subtree:

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
skipped by an ancestor. Skips apply to the entire selected path, including hooks
declared on the same command or a descendant. Unlisted hooks still run. Unknown names are ignored.
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

# Application authentication runs before the inherited project hook:
go run . projects show --token demo --project example
go run . show-project --token demo --project example

# The leaf skips both the application hook and the inherited project hook:
go run . projects public
```

This example accepts any non-empty demo token and keeps it only in memory. It does
not perform real authentication or persist credentials.
