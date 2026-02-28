# Using command aliases

Commands can have aliases, which are specified when creating the commands:

```go
err := app.AddCommand(&naistrix.Command{
	Name:    "run",
	Aliases: []string{"r"},
	// ...
})
```

The `run` command can now be invoked using `r` as an alternative.

Keep in mind that you can not have two commands with the same name or alias on the same command level. If a command has `r` as an alias, you can not have another command with the name `r` or with `r` as an alias on the same level.

## Top-level aliases

One can also register top-level aliases for commands:

```go
err = app.AddCommand(&naistrix.Command{
	Name:  "auth",
	SubCommands: []*naistrix.Command{{
		Name:            "login",
		TopLevelAliases: []string{"login"},
		// ...
	}},
	// ...
})
```

With this setup, the `login` command can be invoked using `cli auth login` or just `cli login`. Note that the top-level alias must be unique across all commands in the application, and can not be the same as any command that exists on the top-level.