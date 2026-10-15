# Application with command flags

An example application showcasing how flags are supposed to be used with Naistrix. The main application can have one or more global ("sticky") flags. These flags will be available for all (sub)commands. Each (sub)command can also have its own set of flags, which will be available for that command. A command can also have its own sticky flags, which also will be available for all subcommands of that command.

## Supported types

The most common types for flags are supported, including, but not limited to:

- `bool`: A boolean flag, can be set to true or false. If only the flag name is provided, it will be set to true. If the flag is not provided by the end user, the value will be set to true. If the default value is true, the end user must set the flag to `false` when running the command.
- `int`: An integer flag, can be set to any integer value.
- `string`: A string flag, can be set to any string value.
- `[]string`: A slice of strings, accepts comma-separated values and repeated flags.
- `naistrix.StringArray`: A slice of strings that preserves each flag argument without splitting on commas. Repeat the flag to provide multiple values, e.g. `--set 'spec.env=[{name: LOG_LEVEL, value: debug}]' --set 'spec.image=my-image'`.
- `time.Duration`: A duration flag, can be set to a duration string (e.g., `1h`, `30m`).
- `naistrix.Count`: A flag that can be repeated to increase a counter. Useful for a "verbose" flag for instance, where `-v` is `1`, `-vv` is `2` and so forth.

## Struct tags

Flags are defined in a struct, and the struct fields can be configured using struct tags. The following tags are supported:

- `name`: The name of the flag. If not specified, the field name will be used.
- `short`: A short version of the flag name, must be a single character.
- `usage`: A text describing the purpose of the flags, used in the help message.

All tags are optional.

## Default values

To set a default value for a flag, simply assign a value to the field in the struct when creating it.

## Completion

Flag types can implement `naistrix.FlagAutoCompleter` or `naistrix.FileAutoCompleter` to provide flag-value completion. Before these callbacks run, Naistrix synchronizes the target command's global, ancestor, and local flag structs from command-line input, configuration, and environment variables. This also applies when a child command completes an inherited global or sticky flag.

Flag-value completion does **not** run `ValidateFlagsFunc`: a required flag may be the very flag the user is trying to complete. Check only the callback's specific prerequisites and return active help if they are missing. For example, completing `--member` may require a selected `--team`, but completing `--team` itself should work before a team is selected.

Before positional argument completion, Naistrix synchronizes all flag structs and runs ancestor-to-child `ValidateFlagsFunc` callbacks. Completion handlers, including `AutoCompleteFunc`, run only if validation succeeds.
