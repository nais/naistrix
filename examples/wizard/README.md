# Command wizards

Set `Command.WizardFunc` to guide users through a command's inputs. The callback
updates the same flag struct used by `RunFunc` and returns positional arguments
in declaration order:

```go
flags := &CreateFlags{Replicas: 2}

command := &naistrix.Command{
    Name: "create", Title: "Create an application", Flags: flags,
    Args: []naistrix.Argument{{Name: "name"}},
    WizardFunc: func(ctx context.Context, _ *naistrix.OutputWriter) ([]string, error) {
        name, err := input.Input(ctx, "Application name")
        if err != nil {
            return nil, err
        }
        if flags.Team == "" {
            flags.Team, err = input.Input(ctx, "Team")
            if err != nil {
                return nil, err
            }
        }
        return []string{name}, nil
    },
    RunFunc: func(_ context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
        out.Printf("Creating %s for %s\n", args.Get("name"), flags.Team)
        return nil
    },
}
```

See [main.go](main.go) for a complete example with validation and conditional
questions. Run it from the repository root in an interactive terminal:

```sh
go run ./examples/wizard create
```

## When it runs

The wizard starts only when no positional arguments or explicit input flags are
supplied. Help, shell completion, and non-interactive runs never start it.
Use `--non-interactive` to skip it even in a terminal.
Supplying input uses the normal command path instead:

```sh
go run ./examples/wizard create my-app --team platform --replicas 2
```

Wizards require a leaf command with `RunFunc` and cannot be combined with
deprecation. Defaults, configuration, and environment variables are loaded before
the callback. Application and command pre-run hooks (including flag-dependent
hook skips) run after flags are synchronized and before the wizard, so they can
prepare authentication or other prerequisites. They see the initial effective
flags, not changes made by the wizard, and are not run again after collection.
Flag validators run afterward, followed by normal argument
resolution and validation. Return all required arguments: missing values cause a
validation error, not additional prompts. Returning an error stops execution.

## Command preview

After validation, Naistrix displays an equivalent invocation on stderr by
default before calling `RunFunc`. The preview includes effective defaults, not
just changed fields. For example, setting a default team skips the example's
team question but still includes `--team platform` in the preview:

```sh
go run ./examples/wizard defaults set team platform
go run ./examples/wizard create
```

Previews use the application's configured name (`wizard-example` here), assuming
an installed executable, rather than reconstructing `go run`. Quoting targets
POSIX-compatible shells, including Bash and Zsh. If an input cannot be faithfully
represented, Naistrix explains why and omits the preview; the command still runs.

Disable the preview with an option from `github.com/nais/naistrix/wizardoption`:

```go
WizardOptions: []wizardoption.Option{
    wizardoption.WithoutCommandPreview(),
},
```

## Operational flags

Mark flags such as verbosity with `wizard:"ignore"` to keep them from suppressing
the wizard or appearing in the preview. They are still applied normally:

```go
Verbose bool `name:"verbose" wizard:"ignore"`
```

This works for local, sticky, and global flags. Naistrix's built-in verbosity,
color, configuration-path, and non-interactive flags already carry the tag.
`--non-interactive` still disables the wizard through the invocation context.
`ignore` is the only supported tag value; other values are initialization errors.

## Sensitive input

Use `input.Secret(ctx, "API token")` for asterisk-masked questions in custom callbacks.
Separately, mark sensitive flag fields or argument definitions for preview
redaction:

```go
Token string `name:"token" sensitive:"true"`

// Positional argument definition:
naistrix.Argument{Name: "secret", Sensitive: true}
```

The preview replaces each sensitive value with `'<REDACTED>'` and tells users to
replace the placeholders. Real values still reach validation and `RunFunc`.
Sensitive arguments also use masked automatic free-form prompts outside the
wizard. Neither metadata nor masking prevents custom callbacks from logging
secrets or hides values in choice lists.
