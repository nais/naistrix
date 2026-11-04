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
        Title:       "Default greeting.",
        Description: "Greeting to use when --greeting is omitted from the greet command.\n\nAn explicit --greeting flag overrides this setting.",
    },
)
```

Keys must be nonempty and contain no whitespace, titles must be nonempty, and keys must be unique (case-insensitively). Keys are displayed in lowercase, matching configuration file keys. Multiple calls to the option append documentation. `Title` is a short summary; `Description` is optional detailed documentation that supports multiple paragraphs. Existing definitions that used `Description` as their summary should move that text to `Title`.

Keep `Title` to a single line and aim for 80 visible characters or fewer so it stays readable in tables and shell completion. Put explanations and examples in `Description`. This is a recommendation, not an enforced length limit; formatting tags do not count toward the suggested length.

At the start of `Application.Run`, each documented key is checked against the registered flag names, including global, command-local, and sticky flags anywhere in the command tree. Unmatched keys return an error before execution. Flags do not all need documentation, and undocumented keys can still be set.

`Application.GenerateDocs` always generates a top-level **Configuration** page at `configuration.md`. It links to the commands for managing settings, respecting a renamed command such as `config`, and includes alphabetical sections for registered keys, each showing its title followed by its description when provided. The keys section is omitted when no key documentation is registered. Local configuration values are never read. In titles and descriptions, matching `<info>...</info>` pairs become Markdown inline code, while matching `<warn>` and `<error>` pairs become bold text. Unmatched or mismatched formatting tags are escaped so they remain visible as literal text. This keeps command placeholders such as `<team>` visible inside inline code. Paragraph breaks and existing Markdown are preserved. Doc generation also validates that documented keys match registered flags.

Titles and descriptions support `<info>`, `<warn>`, and `<error>` formatting tags. `config list` normalizes title whitespace to a single line. Shell completion also strips formatting tags and uses only the title, never the detailed description.

Run `example config list` to see configured values together with documented keys and their short titles in the **Description** column. Documented keys without a configured value show `(not set)`, while configured empty strings show `""`. Documentation does not need to cover every supported key.

Run `example config describe greeting` to see the title, full description with paragraph breaks, and configured value. This command accepts documented keys only, case-insensitively, and offers them in prompts and shell completion. It does not create or modify configuration values.

The **Configured value** column reflects only the configuration file, not environment overrides or built-in flag defaults. When no documentation is registered, the list keeps its two-column format.

With shell completion installed, `example config set <TAB>` suggests documented keys with their titles, together with existing configuration keys. These are suggestions only: undocumented keys remain valid. No value choices or validation are registered.

Try the example:

```sh
go run . config list
go run . config describe greeting
go run . config set greeting Hi
go run . greet World
# Hi, World!
```

`config get` and `config unset` still operate only on values actually stored in the configuration file.
