# Projects: configure and evolve a CLI

Browse a local project catalog using a team from a configuration file or an
explicit flag.

![Projects reads the team from YAML, then lists a different team's projects using an explicit flag](example_0.gif)

## Try it

From the repository root:

```sh
go run ./examples/projects --config examples/projects/config.yaml project ls -v
go run ./examples/projects --config examples/projects/config.yaml list --team platform --offline -v
```

The first invocation lists the `apps` team's projects. The second uses the
top-level alias `list`, overrides the team, and skips the demo authentication
hook. The inherited team hook still runs. `-v` shows which hooks ran on stderr.

Try persisting your own team in a temporary configuration file and generating
Markdown from the same command definitions:

```sh
demo_dir=$(mktemp -d)
go run ./examples/projects --config "$demo_dir/config.yaml" config set team apps
go run ./examples/projects --config "$demo_dir/config.yaml" config describe team
go run ./examples/projects --config "$demo_dir/config.yaml" list
go run ./examples/projects --generate-docs "$demo_dir/docs"
```

Run `go run ./examples/projects show-projects` in a terminal to see a deprecated
command offer to run `project list` in the same invocation.

## In the code

- [`projectCommands`](main.go) groups commands, defines the inherited `--team`
  flag, and registers both kinds of alias. Its offline flag conditionally skips
  the application hook.
- [`run`](main.go) configures the app-wide profile hook, documents `team`, and
  registers the deprecated command. It uses `GenerateDocs` to write the command
  docs and a configuration reference from the same definitions.
