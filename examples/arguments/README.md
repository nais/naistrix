# Using command arguments

Define positional arguments in a command's `Args` field, in the order users should supply them. Access their values by name through `Arguments.Get`, or `Arguments.GetRepeatable` for a repeatable argument, which must be defined last.

## Interactive prompts

In interactive terminals, missing arguments are prompted for automatically. Each argument's optional `Prompt` customizes its question; otherwise a default question is used.

In non-interactive runs (such as CI or piped input/output), missing arguments produce the normal validation error without prompting. Set `Command.DisableArgumentPrompts` to `true` to disable argument prompting for a command.

Empty or whitespace-only answers to text prompts are rejected until the first value is entered.

### Selecting from choices

Set `Choices` to present a selectable list instead of a free-form text prompt:

```go
naistrix.Argument{
    Name:                   "func",
    Prompt:                 "How should the words be transformed?",
    Choices:                []string{"upper", "lower"},
    ChoicesCaseInsensitive: true,
}
```

The user can select with the arrow keys and confirm with Enter, or type to filter the list.

Values supplied on the command line are also validated against `Choices`, even without a `Prompt`.

For dynamic choices that are loaded during command execution, use `ChoicesFunc` instead of `Choices`, as described below.

Matching is case-sensitive by default. Set `ChoicesCaseInsensitive` to accept different casing and normalize values to the spelling in `Choices` before custom validation and command execution.

This applies to every value of a repeatable argument too. Whitespace is not trimmed, and choices that differ only in case cannot be configured together when case-insensitive matching is enabled. Without either `Choices` or `ChoicesFunc`, the option has no effect.

### Repeatable arguments

For repeatable text arguments, enter values one at a time. After the first value, a blank or whitespace-only answer finishes collection. Values containing spaces are preserved as a single argument.

For repeatable arguments with `Choices`, each selection is followed by a confirmation asking whether to add another value. Answer yes to select another value, or no (the default) to finish.

Supplying any values for the repeatable argument on the command line skips interactive collection entirely.

### Dynamic choices

Set an argument's `ChoicesFunc` to load allowed values at execution time. Flag structs have already been synchronized from command-line input, configuration, and environment variables when it runs, so the callback can capture the struct to access the selected team.

The `Arguments` passed to `ChoicesFunc` contain only earlier positional arguments. These values have already been validated and normalized, whether supplied on the command line or prompted for.

#### Validation and execution

Use either `Choices` or `ChoicesFunc` for an argument, not both. `ChoicesFunc` is called at most once for each argument during a command run. Its returned choices are used for both prompting and validation, including all values of a repeatable argument.

`ChoicesFunc` is also called when the user supplies a value on the command line, so that value can be validated against the returned choices. If the argument count is invalid and prompting cannot correct it, the command fails before calling `ChoicesFunc`.

An empty dynamic list means no valid options and produces an error. Errors returned by `ChoicesFunc` and duplicate choices also stop execution. `ChoicesCaseInsensitive` applies to dynamic choices as well, preserving the returned spelling as the canonical value. Custom validation and command execution receive the complete resolved arguments.
