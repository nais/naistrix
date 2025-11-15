# Deprecated commands

Naistrix allows you to deprecate commands in your CLI application. When a command is deprecated, it will still be available for use, but users will receive a warning message indicating that the command is deprecated. Users will be given an option to automatically run the replacement command, if one has been specified.

The original `RunFunc` of the deprecated command will not be executed, so feel free to remove it once the command is deprecated.

Deprecated commands will also be hidden from the help output of the application. The user can still see the help output of the deprecated command when accessing help for that specific command.

## Deprecating a command

To deprecate a command, you can use the `Deprecated` field in the `Command` struct. You can choose to provide a replacement command that users can run instead using a few different functions provided by Naistrix.

### Static replacement

This is the simplest way to provide a replacement for a deprecated command. You can specify a static replacement command using the `DeprecatedWithReplacement` function:

```go
app.AddCommand(&naistrix.Command{
	Name:       "command-v1",
	Title:      "This is the first version of the command",
	Deprecated: naistrix.DeprecatedWithReplacement([]string{"command-v2"}),
	RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error {
		// doing some stuff
		return nil
	},
})
```

The slice passed to `DeprecatedWithReplacement` represents the command and its arguments that will be run as the replacement. The name of the application **must not** be included in the slice, only the replacement command and its args / flags.

### Dynamic replacement

If you need more flexibility in determining the replacement command, you can use the `DeprecatedWithReplacementFunc` function. This function allows you to provide a callback that will be executed when the deprecated command is run. The callback should return a slice of strings representing the replacement command and its arguments.

```go
app.AddCommand(&naistrix.Command{
	Name:       "command-v1", 
	Title:      "This is the first version of the command",
	Deprecated: naistrix.DeprecatedWithReplacementFunc(func(ctx context.Context, args *Arguments) []string {
		// logic to determine replacement command 
		// you can also use the flags for the deprecated command when necessary
		return []string{"command-v2", "--some-flag", "some-value"}
	}),
	RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error {
		// doing some stuff
		return nil
	},
})
```

As with the static replacement, the name of the application **must not** be included in the returned slice, only the replacement command and its args / flags.

### No replacement

If you want to deprecate a command without providing a replacement, you can simply use the `DeprecatedWithoutReplacement` function:

```go
app.AddCommand(&naistrix.Command{
	Name:       "command-v1",
	Title:      "This is the first version of the command",
	Deprecated: naistrix.DeprecatedWithoutReplacement(),
	RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error {
		// doing some stuff
		return nil
	},
})
```