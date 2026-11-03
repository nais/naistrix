# Documentation Generation Example

This example demonstrates how to use Naistrix to generate Markdown documentation for your CLI application.

## Usage

Run the example:

```sh
go run main.go
```

This will generate a set of Markdown files in the `docs/` directory, one for each command and subcommand.

The example registers documentation for the `loud` configuration key using `ApplicationWithDefaultsDocumentation`. The generated `docs/configuration.md` is a top-level **Configuration** page with documented keys and links to the commands for managing settings. Only registered documentation is included, not settings from your local configuration file. Documented keys must match registered flags before docs can be generated. The configuration page is always generated; the keys section is omitted when no key documentation is registered.

The configuration page also explains the default file location on Linux, macOS, and Windows, the `--config <path>` override, and how to see the actual path with `defaults list`. Paths are documented portably rather than embedding the machine-specific path used during generation.

The generated documentation can be used for with instance [Jekyll](https://jekyllrb.com/) to generate HTML documentation for your CLI application.
