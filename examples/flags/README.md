# Application with command flags

An example application showcasing how flags are supposed to be used with Naistrix. The main application can have one or more global ("sticky") flags. These flags will be available for all (sub)commands. Each (sub)command can also have its own set of flags, which will be available for that command. A command can also have its own sticky flags, which also will be available for all subcommands of that command.

## Supported types

The most common types for flags are supported, including, but not limited to:

- `bool`: A boolean flag, can be set to true or false. If only the flag name is provided, it will be set to true. If the flag is not provided by the end user, the value will be set to true. If the default value is true, the end user must set the flag to `false` when running the command.
- `int`: An integer flag, can be set to any integer value.
- `string`: A string flag, can be set to any string value.
- `[]string`: A slice of strings, can be set to multiple values.
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