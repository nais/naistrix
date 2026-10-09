# Using command arguments

An example application using command arguments to control the behavior of the application.

## Interactive prompts

In interactive terminals, missing arguments are prompted for automatically. Each argument's optional `Prompt` customizes its question; otherwise a default question is used.

In non-interactive runs (such as CI or piped input/output), missing arguments produce the normal validation error without prompting. Set `Command.DisableArgumentPrompts` to `true` to disable argument prompting for a command.

Empty or whitespace-only answers to text prompts are rejected until the first value is entered.

## Selecting from choices

Set `Choices` to present a selectable list instead of a free-form text prompt:

```go
naistrix.Argument{
    Name:    "func",
    Prompt:  "How should the words be transformed?",
    Choices: []string{"upper", "lower"},
}
```

The user can select with the arrow keys and confirm with Enter, or type to filter the list.

Values supplied on the command line are also validated against `Choices`, even without a `Prompt`.

## Repeatable arguments

For repeatable text arguments, enter values one at a time. After the first value, a blank or whitespace-only answer finishes collection. Values containing spaces are preserved as a single argument.

For repeatable arguments with `Choices`, each selection is followed by a confirmation asking whether to add another value. Answer yes to select another value, or no (the default) to finish.

Supplying any values for the repeatable argument on the command line skips interactive collection entirely.
