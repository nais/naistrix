# Command wizards

The [example in `main.go`](main.go) uses `Command.WizardFunc` to collect an
application name, a team, and a replica count. The wizard updates the same flag
struct used by `RunFunc` and returns the positional arguments in declaration
order. After validation, `RunFunc` prints what it would create.

Run it from the repository root in an interactive terminal:

```sh
go run ./examples/wizard create
```

Enter an application name and a team if prompted, then press Enter to keep the
current replica count or choose to change it.

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

After validation, Naistrix displays a command on stderr before calling `RunFunc`.
It includes the effective input values, even when the wizard kept a default.
To try this, build the executable with the name used in the preview and set a
default team in a temporary config file:

```sh
demo_dir=$(mktemp -d)
go build -o "$demo_dir/wizard-example" ./examples/wizard
"$demo_dir/wizard-example" --config "$demo_dir/config.yaml" defaults set team platform
"$demo_dir/wizard-example" --config "$demo_dir/config.yaml" create
```

Enter `my-app` and keep the default of two replicas. The wizard skips the team
question and prints this command:

```sh
wizard-example create my-app --replicas 2 --team platform
```

Change the default team, then replay using the executable you built and the same
temporary config file:

```sh
"$demo_dir/wizard-example" --config "$demo_dir/config.yaml" defaults set team other-team
"$demo_dir/wizard-example" --config "$demo_dir/config.yaml" create my-app --replicas 2 --team platform
```

The replay still prints `Would create my-app for team platform with 2 replicas.`
The explicit `--team` and `--replicas` values make it independent of those
defaults. Previews use the application's configured name, even when launched with
`go run`. Quoting targets POSIX-compatible shells, including Bash and Zsh.
If an input cannot be represented, Naistrix explains why and runs the command
without a preview.

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
