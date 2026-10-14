# Using command arguments

Define positional arguments in a command's `Args` field, in the order users should supply them. Access their values by name through `Arguments.Get`, or `Arguments.GetRepeatable` for a repeatable argument, which must be defined last.

## Interactive prompts

In interactive terminals, missing arguments are prompted for automatically. Each argument's optional `Prompt` customizes its question; otherwise a default question is used.

Use `PromptFunc` when the text depends on flags or earlier arguments:

```go
naistrix.Argument{
    Name: "member",
    PromptFunc: func(_ context.Context, args *naistrix.Arguments) (string, error) {
        return fmt.Sprintf("Member of %q to assign the %q role to", flags.Team, args.Get("role")), nil
    },
}
```

Flags are synchronized before the callback runs, and `args` contains only earlier, validated and normalized positional arguments. `Prompt` and `PromptFunc` are mutually exclusive; configuring both is an error. An empty callback result uses the normal default question when prompting and adds no helper text during completion. The same resolved text is used as active help during automatic choice completion.

The callback runs only when its argument needs prompting or completion help, at most once per argument per run or completion request. Supplied arguments do not trigger it during execution. Text is resolved afresh each time without changing the command definition. Keep callbacks cheap and side-effect-free; avoid API calls or printing output just to render text.

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

## Shell completion

After enabling shell completion for the application, `Choices` and `ChoicesFunc` automatically suggest values for the argument being entered. Choices are filtered by the typed prefix and remain available for additional values of a repeatable argument, including values already entered.

When the current argument has a `Prompt` or `PromptFunc`, automatic choice completion also displays the resolved text as shell active help. For example, the `func` argument above shows "How should the words be transformed?" alongside its choices. Arguments without custom text do not add help text; default interactive questions are not used for completion.

`AutoCompleteExtensions` takes priority, followed by a command's explicit `AutoCompleteFunc`. Automatic choice completion is only used when neither override is configured.

For dynamic choices, earlier arguments are validated and normalized, and flag structs are synchronized before calling `ChoicesFunc`. As during execution, the callback sees only earlier positional arguments, not values already entered for the current repeatable argument. Each completion request resolves a fresh list; callbacks should use the supplied context and avoid printing to standard output.

Case-insensitive completion preserves the prefix as typed so shells can match suggestions. Completed values are normalized to the configured spelling during command execution, just like supplied values.

Completion never prompts or runs custom validation or the command itself. If a callback fails, returns no choices, or returns ambiguous choices, completion displays the diagnostic as shell active help and also writes it to standard error. It supplies no selectable suggestions and does not fall back to file suggestions. Multiline diagnostics are flattened to a single line for active help.
