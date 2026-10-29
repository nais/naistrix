# ADR-001: Command wizards

## Status

Proposed.

## Context

Naistrix already prompts for missing positional arguments, but some commands need
a guided workflow that also collects flags and asks conditional questions.
Developers should be able to reuse their typed flag structs rather than maintain
a second input model or manually construct CLI tokens using flag names.

Users should also learn how to automate the same operation. An equivalent command
preview must preserve effective inputs, including configured defaults, without
exposing secrets or silently relying on defaults that may change later.

## Decision

An optional wizard collects a command's inputs. It updates the same flag struct
that the command normally uses and returns positional arguments.
Naistrix then validates those inputs, displays an equivalent CLI invocation, and
calls the ordinary `RunFunc` once. It does not start a subprocess or recursively
execute the CLI.

- Opt in per leaf command using `WizardFunc`, with `RunFunc` still required.
- Trigger only in interactive terminals with no positional arguments or explicitly
  supplied input-affecting flags. Use `wizard:"ignore"` for operational flags.
- Treat returned arguments as complete input: validate them without falling back
  to automatic missing-argument prompts.
- Enable a POSIX-shell command preview by default, using final effective inputs.
  Allow opt-out with `wizardoption.WithoutCommandPreview()`.
- Redact sensitive flags and arguments. Omit the entire preview if an input cannot
  be faithfully represented on the CLI, without blocking valid execution.
- Keep flag structs ordinary structs, without assignment or provenance tracking.

### Scope

This decision covers custom wizards for executable leaf commands. It does not
introduce generated forms, automatic wizards for partially supplied input, a
force-wizard CLI flag, or a separate command execution path. Commands without a
wizard retain their existing behavior.

## Alternatives considered

- **Only use existing missing-argument prompts.** This remains useful for simple
  commands, but does not collect flags or support a developer-controlled workflow
  spanning several inputs.
- **Generate a form from every flag.** Flag metadata provides names, types, and
  defaults, but does not describe question order, conditional steps, or which
  optional inputs are worth asking about. A custom callback leaves those choices
  with the application developer.
- **Have the callback return CLI tokens.** This naturally supports parsing and
  previews, but requires developers to duplicate flag names and convert typed
  answers to strings. Mutating the existing struct keeps field references
  compiler-checked; Naistrix owns flag-name lookup and serialization.
- **Re-execute Cobra or start a subprocess.** This would reuse CLI parsing, but
  complicates execution hooks, configuration, cancellation, and prevention of
  recursive wizard entry. Continuing through validation and the existing
  `RunFunc` avoids a second execution.

## API examples

### Defining a wizard

The callback controls the questions, their order, and any conditional steps. There
is no automatically generated questionnaire for the command's flags.

The callback type is:

```go
type WizardFunc func(ctx context.Context, out *OutputWriter) ([]string, error)
```

`Command.WizardFunc` holds this callback; `Command.WizardOptions` holds a slice of
`wizardoption.Option`. The callback captures the same flag struct pointer as
`RunFunc`, as illustrated below.

```go
package main

import (
    "context"
    "strings"

    "github.com/nais/naistrix"
    "github.com/nais/naistrix/input"
)

type CreateFlags struct {
    Team     string `name:"team" usage:"Team that owns the application."`
    Replicas int    `name:"replicas" usage:"Number of application replicas."`
}

func createCommand() *naistrix.Command {
    flags := &CreateFlags{Replicas: 2}

    return &naistrix.Command{
        Name:  "create",
        Title: "Create an application",
        Flags: flags,
        Args:  []naistrix.Argument{{Name: "name"}},

        WizardFunc: func(_ context.Context, _ *naistrix.OutputWriter) ([]string, error) {
            name, err := input.Input("Application name")
            if err != nil {
                return nil, err
            }

            // Effective defaults have already been loaded into flags. Only ask
            // for a team if the user does not already have one configured.
            if flags.Team == "" {
                team, err := input.Input("Team that owns the application")
                if err != nil {
                    return nil, err
                }
                flags.Team = strings.TrimSpace(team)
            }

            // Keep the effective replica count unless the user wants to change it.
            changeReplicas, err := input.Confirm("Customize the replica count?")
            if err != nil {
                return nil, err
            }
            if changeReplicas {
                replicas, err := input.Select("Number of replicas", []int{1, 2, 3})
                if err != nil {
                    return nil, err
                }
                flags.Replicas = replicas
            }

            // Return argument values in the order declared in Args. Flag names
            // and CLI serialization are Naistrix's responsibility.
            return []string{strings.TrimSpace(name)}, nil
        },

        ValidateFlagsFunc: func(context.Context) error {
            if strings.TrimSpace(flags.Team) == "" {
                return naistrix.Errorf("--team is required")
            }
            if flags.Replicas < 1 {
                return naistrix.Errorf("--replicas must be at least 1")
            }
            return nil
        },

        ValidateArgsFunc: func(_ context.Context, args *naistrix.Arguments) error {
            if strings.TrimSpace(args.Get("name")) == "" {
                return naistrix.Errorf("application name must not be empty")
            }
            return nil
        },

        RunFunc: func(_ context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
            // Real application creation belongs here, not in WizardFunc.
            out.Printf("Creating %s for team %s with %d replicas\n",
                args.Get("name"), flags.Team, flags.Replicas)
            return nil
        },
    }
}
```

Both callbacks capture the same `flags` pointer. Renaming a field is checked by
the compiler; changing a flag's `name` tag automatically changes the preview.
The wizard does not construct flag tokens or maintain a separate input struct.

## Implementation notes

### When the wizard runs

- A command opts in by providing `WizardFunc`.
- Initially, only leaf commands support wizards; `RunFunc` remains required.
  Combining a wizard with `SubCommands` or `Deprecated`, or setting
  `WizardOptions` without `WizardFunc`, is a command-definition error.
- The wizard starts automatically in an interactive terminal when no positional
  arguments or explicit input-affecting flags were supplied. Both standard input
  and standard output must be terminals, matching `input.IsInteractive()`.
- Operational flags, such as verbosity, do not suppress the wizard. Explicit
  input-affecting flags do, including inherited flags and explicitly supplied
  zero values such as `--enabled=false`.
- Defaults, configuration, and environment variables do not count as explicitly
  supplied CLI flags.
- Help and shell completion never start the wizard.
- Non-interactive execution retains normal validation and error behavior.
- Cancelling or returning an error stops execution without calling `RunFunc`.
- The wizard takes precedence over ordinary missing-argument prompting. Commands
  without a wizard, and invocations that do not trigger one, retain their existing
  argument-prompt behavior. `DisableArgumentPrompts` controls those ordinary
  prompts; it does not disable an explicitly configured wizard.
- The wizard's returned argument slice is its complete positional input. Missing
  arguments produce the normal argument-count validation error, without falling
  back to automatic prompting. Validation failure prevents both the preview and
  `RunFunc`.

For example, assuming this command is registered under `example app`:

```sh
example app create                               # Wizard in an interactive terminal.
example app create my-app --team platform        # Normal execution.
example app create --team platform               # No wizard; normal argument handling.
example app create --help                        # Help only.
```

#### Ignoring operational flags

Use `wizard:"ignore"` on operational flags, such as verbosity, that should not
affect wizard triggering or appear in the command preview:

```go
type GlobalFlags struct {
    Team    string `name:"team" usage:"Team to use."`
    Verbose bool   `name:"verbose" usage:"Enable verbose output." wizard:"ignore"`
}
```

- Without the tag, an explicitly supplied flag suppresses the wizard, and its
  effective value is included in the preview when a wizard runs.
- With `wizard:"ignore"`, explicitly supplying the flag does not suppress the
  wizard, and its value is omitted from the preview. The flag is still parsed
  and applied normally.
- `ignore` is initially the only supported explicit value. Any other value,
  including an explicitly empty tag, is a command-definition error during
  initialization rather than being silently ignored.

These rules apply to local, sticky, and global flags. Naistrix classifies
Naistrix's built-in `--verbose`, `--no-colors`, and `--config` flags as operational
and gives them the tag too. Application developers classify their own flags.
Help and shell completion retain their special non-wizard behavior.

```sh
example app create --verbose         # Wizard; verbose output still applies.
example app create --team platform   # No wizard; normal argument handling.
```

The tag couples wizard triggering and preview omission; separate controls are
outside the initial scope. It is not a security mechanism for sensitive values.

### Execution order

Wizard execution follows these steps:

1. Parse flags and synchronize effective values from defaults, configuration, and
   environment variables into the flag structs.
2. Run the wizard before flag or argument validators that could reject inputs the
   wizard is meant to collect, including applicable ancestor validators.
3. Validate the final flags ancestor-first, then resolve and validate arguments
   for applicable ancestors and the target command using their normal count,
   choice, normalization, and custom checks. Automatic missing-argument prompting
   is disabled for this step.
4. Display the equivalent invocation, unless disabled.
5. Call `RunFunc` once with the resolved arguments and updated flag struct.

Wizard assignments must survive this process: no later synchronization may
overwrite them with the pre-wizard configuration. The preview uses the validated,
normalized inputs. Validation failure prevents both the preview and `RunFunc`.
Ordinary invocations and shell completion retain their existing validation order.

### Automatic command preview

Suppose the user's Naistrix defaults already select team `platform`. The wizard
above skips the team question, the user enters `my-app`, and keeps two replicas.
The preview still includes both effective flag values:

```text
To run this non-interactively next time:

  example app create my-app --team platform --replicas 2
```

The preview captures effective inputs, not just fields changed by the wizard or
values that differ from declared defaults. Changing the user's default team or
replica count later should not change those inputs when the preview is reused.
This applies to input-affecting inherited flags too. Flags marked
`wizard:"ignore"` are omitted. False booleans, zero numeric values, and empty
strings are inputs too; they must not be silently omitted as though unspecified.

Naistrix is responsible for flag names, shell quoting, explicit boolean values,
repeatable-flag semantics, and separating positional arguments from flags when
needed. The callback returns raw argument values, not shell-quoted strings.

The preview uses the application's configured executable name and canonical
command path, including when invoked through an alias. It assumes that executable
is available under its configured name; it does not reconstruct `go run`, shell
wrappers, or the path used to launch the program.

#### Shell syntax

The preview targets POSIX-compatible shell syntax, including Bash and Zsh.
Naistrix quotes and escapes individual tokens so spaces, quotes, and shell
metacharacters are preserved as literal input rather than interpreted as shell
expressions. It does not detect the current shell or select a format using
`$SHELL`. PowerShell and Windows command-prompt syntax are outside the initial
scope.

The preview is an equivalent invocation, not a guarantee of identical results
across environments or changes to external resources. If an input cannot be
faithfully represented, Naistrix must explain that limitation or omit the preview
rather than display a misleading command. Preview limitations must not prevent
otherwise valid command execution.

#### Values without a faithful CLI representation

If any effective input cannot be faithfully represented on the command line,
Naistrix skips the entire preview rather than displaying a partial or misleading
invocation. It explains the limitation without exposing the value, identifying
the flag where applicable:

```text
Command preview unavailable: --resources cannot represent an empty override.
```

For example, a wizard might clear a configured list whose flag parser has no way
to express an empty override. This decision does not introduce new flag-parsing
semantics just to support previews. The validated command still executes normally.
When preview is explicitly disabled, no preview-unavailable message is needed.

Sensitive-value redaction is different: the preview remains available with an
explicit instruction to replace its placeholders.

#### Disabling the preview

Preview is enabled by default. A command can opt out with the options API:

```go
import "github.com/nais/naistrix/wizardoption"

// Fields on the command definition:
WizardOptions: []wizardoption.Option{
    wizardoption.WithoutCommandPreview(),
},
```

Disabling the preview does not change prompting, validation, or execution.

#### Sensitive flag values

Mark sensitive flag fields with `sensitive:"true"`:

```go
type CreateFlags struct {
    Team  string `name:"team" usage:"Team that owns the application."`
    Token string `name:"token" usage:"API token." sensitive:"true"`
}
```

An absent tag or `sensitive:"false"` leaves the field non-sensitive. Other explicit
values are command-definition errors. Application authors must mark sensitive
fields; Naistrix does not infer sensitivity from names or inspect values for secrets.

The preview retains the flag name but replaces its entire value with the fixed
placeholder `'<REDACTED>'`. It never prints part of the original value or reveals
its length. This applies regardless of whether the value came from the wizard,
configuration, environment variables, or defaults. Repeated sensitive values
must also be redacted individually.

```text
To run this non-interactively next time, replace the redacted values:

  example app create my-app --team platform --token '<REDACTED>'
```

Redaction affects only the preview. Validation and `RunFunc` receive the real
value. Unlike `wizard:"ignore"`, `sensitive:"true"` does not change wizard
triggering: explicitly supplying a sensitive input flag still suppresses the
wizard. If both tags are present, `wizard:"ignore"` omits the flag entirely.

#### Sensitive positional arguments

Mark sensitive positional arguments with `Argument.Sensitive`:

```go
Args: []naistrix.Argument{
    {Name: "name"},
    {Name: "secret", Sensitive: true},
},
```

The preview preserves argument order but replaces each sensitive value with
`'<REDACTED>'`, with the same warning about replacing placeholders:

```sh
example app create my-app '<REDACTED>' --team platform
```

For a repeatable sensitive argument, every value is redacted. Argument resolution,
validation, and `RunFunc` still receive the real values. Sensitivity does not
change wizard triggering or require disabling the entire preview.

#### Security boundaries

Sensitivity metadata protects values in the generated preview, not all output
from the application. It is not a general redaction policy for help, validation
errors, logs, or custom callback output. Sensitive choice lists are not masked;
automatic masking applies only to free-form argument prompts.

Redacted previews require editing before use. Supplying real secrets as CLI
arguments can expose them through shell history or process arguments, regardless
of preview redaction or masked interactive input.

### Flag values and explicit CLI input

Flag structs remain ordinary structs, with no tracking fields or provenance API.
Validators and `RunFunc` use the final effective values, regardless of whether
they came from configuration, defaults, CLI input, or wizard assignments.

Assigning a field in the wizard does not mark its flag as explicitly supplied in
the underlying flag parser's bookkeeping. Checks for required input should
validate the effective struct value, not require that the flag appeared on the
command line. The preview likewise uses effective values, not parser presence
markers.

The wizard does not attempt to track which fields the callback touched; assigning
the existing value is indistinguishable from leaving the field unchanged.

## Consequences

### Benefits

- Interactive and non-interactive execution share validation and business logic.
- Developers reuse typed flag fields without duplicating flag names in CLI tokens.
- Previews help users automate operations while preserving effective defaults.
- Sensitivity metadata protects library-generated previews, and masked input
  hides the actual characters when collecting secrets.
- Existing invocations without a wizard retain their argument-prompt and
  validation behavior.

### Tradeoffs and limitations

- Developers write custom prompts and explicitly choose masked input inside wizard
  callbacks; there is no generated form or automatic inference of assignments.
- Wizard execution requires validation to happen after interactive collection,
  including applicable ancestor validators, without overwriting collected values.
- Faithful previews require type-aware serialization and shell escaping. Some
  input states cannot be represented and therefore have no preview.
- Redacted previews require editing, and POSIX previews do not target PowerShell
  or the Windows command prompt.
- Sensitivity metadata does not prevent secrets from being printed by custom
  callbacks or exposed elsewhere in the application.
- Previews preserve input values, not external state or identical behavior across
  environments. Dependencies outside the represented inputs, such as other
  configuration data or credentials, are not captured. Flag assignment provenance
  is not available.
