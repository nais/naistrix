# Documentation Generation Example

This example demonstrates how to use Naistrix to generate Markdown documentation for your CLI application.

## Usage

Other angle-bracket tags, including placeholders such as `<team>` and `<env>`, are escaped in prose so browsers display them literally. Inside `<info>` code spans, placeholders are kept as-is because Markdown inline code already protects them from HTML interpretation.

Generated command descriptions and local/inherited flag descriptions use the same formatting conversion: matching `<info>...</info>` pairs become Markdown inline code, while matching `<warn>` and `<error>` pairs become bold text. Unmatched or mismatched formatting tags are escaped so they remain visible as literal text. As with terminal color formatting, tags must not be nested. Formatting tags in example descriptions are stripped only when paired because those descriptions appear as comments inside shell code blocks. Example commands are left unchanged.

Run the example:

```sh
go run main.go
```

This will generate a set of Markdown files in the `docs/` directory, one for each command and subcommand.

The example registers documentation for the `loud` configuration key using `ApplicationWithDefaultsDocumentation`. The generated `docs/configuration.md` is a top-level **Configuration** page with documented keys and links to the commands for managing settings, including `defaults describe <key>`. Each key has a plain-text heading and shows its short `Title` followed by its optional detailed `Description`. Paragraph breaks and existing Markdown are preserved; terminal `<info>` tags become inline code, while `<warn>` and `<error>` tags become bold text. Only registered documentation is included, not settings from your local configuration file. Documented keys must match registered flags before docs can be generated. The configuration page is always generated; the keys section is omitted when no key documentation is registered.

The configuration page also explains the default file location on Linux, macOS, and Windows, the `--config <path>` override, and how to see the actual path with `defaults list`. Paths are documented portably rather than embedding the machine-specific path used during generation.

The generated documentation can be used for with instance [Jekyll](https://jekyllrb.com/) to generate HTML documentation for your CLI application.
