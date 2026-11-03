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

## Document configuration keys

Use `ApplicationWithDefaultsDocumentation` to explain settings:

```go
naistrix.ApplicationWithDefaultsDocumentation(
    naistrix.DefaultDefinition{
        Key:         "greeting",
        Description: "Greeting to use when --greeting is omitted from the greet command.",
    },
)
```

Keys must be nonempty and contain no whitespace, descriptions must be nonempty, and keys must be unique (case-insensitively). Keys are displayed in lowercase, matching configuration file keys. Multiple calls to the option append documentation.

At the start of `Application.Run`, each documented key is checked against the registered flag names, including global, command-local, and sticky flags anywhere in the command tree. Unmatched keys return an error before execution. Flags do not all need documentation, and undocumented keys can still be set.

`Application.GenerateDocs` always generates a top-level **Configuration** page at `configuration.md`. It links to the commands for managing settings, respecting a renamed command such as `config`, and includes an alphabetical list of registered keys and descriptions when available. The keys section is omitted when no key documentation is registered. Local configuration values are never read. Descriptions, including formatting tags, are passed through unchanged, as with other generated documentation. Doc generation also validates that documented keys match registered flags.

Descriptions support `<info>`, `<warn>`, and `<error>` formatting tags in `config list`. Shell completion strips these tags and displays descriptions as single-line plain text.

Run `example config list` to see stored values together with documented keys and their descriptions. Documented keys without a stored value show `(not set)`, while stored empty strings show `""`. Documentation does not need to cover every supported key.

The **Stored value** column reflects only the configuration file, not environment overrides or built-in flag defaults. When no documentation is registered, the list keeps its two-column format.

With shell completion installed, `example config set <TAB>` suggests documented keys with their descriptions, together with existing configuration keys. These are suggestions only: undocumented keys remain valid. No value choices or validation are registered.

Try the example:

```sh
go run . config list
go run . config set greeting Hi
go run . greet World
# Hi, World!
```

`config get` and `config unset` still operate only on values actually stored in the configuration file.
