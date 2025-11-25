# ADR-002: Command wizards

## Status

Accepted.

## Context

Consider a command that creates an application. Naistrix can already ask for a
missing application name, but it doesn't know whether to ask for `--team` when
the user has a team configured, or whether the user wants to change the replica
count. The command author knows which questions to ask and when.

Someone who creates an application this way should be able to take the resulting
command from the terminal and put it in a script. If the wizard used two replicas
and the configured team was `platform`, the script should say so explicitly.
It shouldn't change meaning when the user's defaults change.

## Decision

A leaf command can define a `WizardFunc` alongside its `RunFunc`:

```go
type WizardFunc func(ctx context.Context, out *OutputWriter) ([]string, error)
```

The wizard reads and updates the command's flag struct and returns positional
arguments in declaration order. It collects input; `RunFunc` still performs the
operation. The [deploy example](../examples/deploy/) shows how the two callbacks
share the flag struct. Wizards are not available on parent or deprecated
commands, and wizard options require a wizard callback.

### When to ask questions

A bare invocation such as `example app create` starts the wizard if both stdin
and stdout are terminals and the invocation permits interaction. Supplying a
positional argument or an input flag, even an inherited flag or an explicit
`--enabled=false`, takes the normal path. Defaults, configuration, and
environment variables don't count as flags supplied on the command line.

A flag tagged `wizard:"ignore"` is still parsed, but doesn't stop the wizard or
appear in its preview. That lets `example app create --verbose` ask the same
questions as the bare command. The built-in verbosity, color, and config flags
use this tag. So does `--non-interactive`, which stops the wizard through the
invocation's interaction policy, just like `input.WithNonInteractive(ctx)`.
`ignore` is the only supported value for the tag.

Help and shell completion never open the wizard. `DisableArgumentPrompts`
controls the ordinary missing-argument prompts, not the wizard. Invocations
that skip the wizard keep their normal prompting and validation behavior.

### Where it runs

Naistrix loads the effective flag values, evaluates hook skips, and runs the
applicable [pre-run hooks](001-pre-run-hooks.md) before calling the wizard. That
lets a hook authenticate before a question needs to fetch choices. Hooks and
their skip conditions see the pre-wizard values and aren't run again.

On return, Naistrix validates the final flags from ancestor to leaf, then
resolves and validates the arguments. The wizard's answer is complete: if it
leaves out a required argument, validation fails rather than asking a follow-up
question. A hook or wizard error, cancellation, or failed validation stops
execution. Otherwise Naistrix prints the preview, if enabled, and calls
`RunFunc` once.

Naistrix doesn't reload flag values after the wizard or mark its assignments as
explicit CLI flags. Required-value checks should read the final struct fields,
not ask whether the parser saw a flag on the command line. The application
remains single-use even if collection or validation fails.

### What the preview promises

The preview goes to stderr by default so it doesn't mix with command results.
It uses the configured executable name and the canonical command path, even
when the user came in through an alias. For the example above it might show:

```sh
example app create my-app --replicas 2 --team platform
```

The command includes normalized arguments and final input flag values, including
inherited flags, defaults, and values such as `false`, `0`, and the empty string.
Naistrix handles repeatable flags and quotes values for POSIX shells. If there
are no arguments or input flags to keep a replay out of the wizard, the preview
adds `--non-interactive`.

If the CLI can't express a value, Naistrix explains why it can't show a preview
and runs the command anyway. Clearing a list whose flag parser doesn't accept
an empty override is one example. A command can also turn off the preview with
`wizardoption.WithoutCommandPreview()` in `Command.WizardOptions`.

Mark flag fields with `sensitive:"true"` and positional arguments with
`Argument.Sensitive` to replace each sensitive value with a `<REDACTED>`
placeholder. The preview tells the user to replace the placeholders before
running it. Validation and `RunFunc` still get the original values. A sensitive
flag still counts as explicit CLI input unless it also has
`wizard:"ignore"`, in which case it doesn't appear in the preview at all.

Sensitive arguments use masked input for ordinary free-form prompts. For their
own wizard questions, authors can call `input.Secret`. Neither mechanism hides
choice lists or protects secrets printed in help, errors, or application output.
Replacing a placeholder with a real secret can expose it in shell history or
process arguments.

## Alternatives considered

- **Extend the existing argument prompts.** That works for a missing name, but
  a prompt attached to one argument isn't a natural place to collect flags.
- **Generate a form from the flags.** The flag definitions don't say which
  questions can be skipped or which answers determine what to ask next.
- **Return CLI tokens and run the parser again.** That makes the preview easy to
  print, but the callback must spell flag names and turn typed answers back into
  strings. Restarting command execution would also mean guarding against running
  hooks twice or reopening the wizard. Serialize for display, not execution.

## Consequences

Command authors write the questions, but only one implementation validates and
performs the operation. Naistrix takes on the work of rendering a command that
the parser will read back correctly. The preview captures inputs, not external
state such as credentials or resources, and it doesn't target PowerShell or
Windows command-prompt syntax. This decision doesn't add generated forms, a
force-wizard flag, or a wizard that fills in partially supplied input.
