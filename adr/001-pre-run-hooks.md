# ADR-001: Pre-run lifecycle hooks

## Status

Proposed.

## Context

Naistrix resolves dynamic argument choices and prompts for missing arguments
before calling a command's `RunFunc`. Applications such as `nais/cli` may need
authenticated API access to produce those choices or otherwise prepare prompts.
Starting authentication inside `RunFunc` is therefore too late.

Authentication is an application-wide concern, but some commands must remain
available without it. In particular, login commands must be able to start their
own authentication flow without first passing an authentication prerequisite.
Other setup concerns may apply to those commands even when authentication does
not, so opting out must be selective rather than disabling all hooks.

Setup also needs effective configuration and flag values, including inherited
flags and the selected command's local flags. Help and shell completion must not
unexpectedly launch an interactive login flow.

## Decision

Introduce named, application-wide pre-run hooks through
`ApplicationWithPreRunHooks`. Each `PreRunHook` has a unique, non-blank `Name` and
a required `RunFunc` of type:

```go
type PreRunHookFunc func(ctx context.Context, out *OutputWriter) error
```

Hooks run in registration order after configuration and flags on the entire
selected command path have been synchronized, but before Naistrix invokes any
flag validators, resolves argument choices, prompts for arguments, or validates
argument values. This makes authentication available to both ancestor callbacks
and the selected command's input handling.

Hooks receive only the execution context and the application's output writer.
They do not receive a command definition: exposing mutable callbacks such as
`RunFunc` would invite bypassing validation, mutating execution behavior, or
executing the command twice. Hooks perform setup; Naistrix owns command execution,
and hook exceptions are declared through opt-outs rather than command-name checks.
They also do not receive resolved positional arguments: producing those arguments
is work the hook may need to enable. An error stops execution immediately and is
returned unchanged.

Commands opt out of individual hooks by name through `SkipPreRunHooks`. Opt-outs
apply to the command and its entire subtree. Descendant opt-outs are additive:
they can skip more hooks, but cannot re-enable a hook skipped by an ancestor.
Unlisted hooks still run in their original order. Unknown skip names are ignored.

The built-in `defaults` subtree can opt out through
`ApplicationWithDefaultsCommandSkipPreRunHooks`. This sets the same skip metadata
on the internally created parent command without exposing a mutable built-in
definition. It applies even when the command is renamed. Help bypasses hooks
automatically and needs no explicit opt-out.

### Scope

This decision introduces one early pre-run phase, not a general middleware
framework. It does not add post-run hooks, deferred cleanup, command-local hook
registration, or completion hooks. Credential storage, token refresh, login
mechanisms, and decisions about interactive authentication remain the
application's responsibility.

## Alternatives considered

- **Authenticate inside `RunFunc` or a wrapper around it.** This is too late for
  dynamic choices, argument prompts, and ancestor validation callbacks.
- **Use `ValidateFlagsFunc` for authentication.** It runs early enough for some
  input handling, but mixes setup and input validation, requires application-wide
  policy to be distributed among commands, and participates in positional
  completion. Authentication may be interactive and should not start on tab
  completion.
- **Expose Cobra's persistent pre-run hooks directly.** That would leak the
  underlying command framework and require application authors to coordinate
  synchronization, ancestor traversal, aliases, and generated commands with
  Naistrix's own hooks. Naistrix should own that ordering contract.
- **Authenticate before calling `Application.Run`.** The selected command and its
  effective flags are not yet available through the normal execution pipeline.
  Identifying exceptions would require duplicating command parsing and alias
  resolution, including special cases for help and completion.
- **Use a single anonymous hook or a skip-all boolean.** This is simpler, but
  prevents commands from bypassing authentication while retaining unrelated setup
  such as auditing. Names allow independent hooks and targeted exceptions.
- **Support a full lifecycle or middleware API immediately.** Additional phases,
  cleanup semantics, and context replacement are not needed to solve early
  authentication and would introduce contracts without a concrete use case.
- **Expose mutable built-in commands or introduce general hook-targeting rules.**
  Either could support exceptions for built-ins, but exposes broader customization
  or adds a second policy mechanism. A defaults-specific option reuses inherited
  skips and solves the current need without new command identity or path APIs.
- **Pass the selected `Command` to hooks.** This could support command-specific
  setup, but exposes mutable definitions and execution callbacks unnecessarily.
  Declarative skips cover the current need. If a concrete use case later requires
  command information, read-only invocation metadata can be considered separately.

## API examples

Register authentication when creating the application. The application supplies
`ensureAuthenticated`, which can reuse credentials, refresh them, or start login
when appropriate:

```go
app, _, err := naistrix.NewApplication("nais", "Nais CLI", "v1.0.0",
    naistrix.ApplicationWithPreRunHooks(naistrix.PreRunHook{
        Name: "auth",
        RunFunc: func(ctx context.Context, out *naistrix.OutputWriter) error {
            return ensureAuthenticated(ctx, out)
        },
    }),
    naistrix.ApplicationWithDefaultsCommandSkipPreRunHooks("auth"),
)
```

An authentication subtree can manage its own login flow. Other registered hooks
still run, including when a subcommand is invoked through a top-level alias:

```go
auth := &naistrix.Command{
    Name: "auth",
    Title: "Authentication commands",
    SkipPreRunHooks: []string{"auth"},
    SubCommands: []*naistrix.Command{
        {
            Name: "login",
            Title: "Log in",
            TopLevelAliases: []string{"login"},
            RunFunc: login,
        },
        {Name: "logout", Title: "Log out", RunFunc: logout},
    },
}
```

A runnable demonstration is available in [examples/hooks](../examples/hooks/).

## Implementation notes

### Execution ordering

1. Cobra selects the command and parses flags. Parsing errors and explicit help
   or version requests retain their normal behavior.
2. Naistrix loads configuration and synchronizes global flags and all applicable
   command flag structs, ancestor-first. A failure prevents hooks from running.
3. Naistrix collects skipped hook names from the selected command and its
   ancestors, then runs the remaining hooks in registration order.
4. The normal ancestor and target validation and argument-resolution pipeline
   continues. Hooks have already completed before any of these callbacks run.
5. On successful input resolution and validation, Naistrix calls `RunFunc`.

Flag synchronization must not be repeated after hooks in a way that overwrites
state they established. Registering hooks is optional; applications without hooks
retain their normal command execution behavior, with synchronization of the full
command path moved ahead of validation.

### Special command paths

- Help, version output, shell completion, completion script generation, and
  parent commands that only display subcommands do not run hooks. This includes
  both positional and flag-value completion.
- Regular aliases use the selected command's hooks and opt-outs. Top-level alias
  wrappers do not run hooks themselves; the resolved target runs them once and
  retains opt-outs from its original ancestors.
- Deprecated commands run hooks before resolving arguments and displaying a
  replacement prompt. Accepting a replacement is a separate command execution,
  with that command's own hooks and opt-outs.
- Built-in `defaults` subcommands run application hooks by default. Setting
  `ApplicationWithDefaultsCommandSkipPreRunHooks("auth")` exempts the entire
  subtree from that hook while retaining others. Multiple skip options append
  names, with the same unknown-name behavior as `Command.SkipPreRunHooks`.
  Renaming the built-in command does not affect its skips, regardless of option
  order. Unrelated commands with matching leaf names are unaffected.
- Each `Application.Run` call runs applicable hooks again. Registration order is
  preserved across multiple `ApplicationWithPreRunHooks` options.

### Failure and interaction contracts

Duplicate or blank hook names and nil hook functions are application-definition
errors. Skipping hooks does not bypass flag or argument validation.

Naistrix does not make hooks interactive-only. They also run when all arguments
are supplied or execution is non-interactive. Authentication hooks should check
existing credentials before prompting and handle non-interactive use according
to the application's policy. Returning a cancellation or authentication error
prevents later hooks, input handling, and command execution.

Completion does not run pre-run hooks or initiate interactive login. Applications
should handle credential reuse and non-interactive refresh in their shared
API/authentication layer, and propagate authentication failures through completion
diagnostics. This avoids duplicating authentication checks in individual
completion callbacks.

## Consequences

### Benefits

- Applications can enforce authentication before input handling needs protected
  resources, without repeating setup across commands.
- Hooks can use effective global, inherited, and local flags.
- Login and other public commands can bypass authentication while retaining
  unrelated setup, with consistent behavior through aliases.
- Help and completion remain free of unexpected hook-driven interaction.
- The public API remains independent of Cobra's hook traversal details.
- The built-in defaults command can bypass authentication without exposing its
  implementation or relying on ambiguous leaf-name checks inside hooks.

### Tradeoffs and limitations

- Hook names are strings, so opt-out typos are not compiler-checked and unknown
  names are silently ignored. Shared constants can reduce this risk.
- Inherited skips cannot be reversed by descendants; command trees must place
  exceptions at an appropriate level.
- Hooks run before semantic input validation, so authentication may occur even
  when a command will later fail validation.
- Hooks may run repeatedly in one process. Authentication implementations must
  avoid unnecessary repeated login and manage their own credential state.
- Completion relies on the shared API/authentication layer for credential reuse
  and non-interactive refresh, rather than hook-driven interactive login.
- The defaults-specific option adds a narrow API; future built-in commands may
  need their own customization options if similar use cases arise.
- Opt-outs are application policy, not a replacement for authorization checks
  in protected services. Application authors must review which commands bypass
  authentication and avoid exposing credentials through hook output.
