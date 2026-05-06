# Custom defaults command name

Naistrix automatically adds a built-in command for managing default flag values, named `defaults`. Use `ApplicationWithDefaultsCommandName` to rename it to something that better fits your CLI's vocabulary, such as `config`, `settings`, or `prefs`.

```go
app, _, err := naistrix.NewApplication(
    "example",
    "Example application",
    "v0.0.0",
    naistrix.ApplicationWithDefaultsCommandName("config"),
)
```

With the option above, users would run `example config set <key> <value>` instead of `example defaults set <key> <value>`.
