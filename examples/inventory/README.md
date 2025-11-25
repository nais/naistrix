# Inventory: output for people and scripts

Browse a local service catalog, export the same records, or look up a service
and handle a machine-readable failure.

![Inventory lists services in a table and returns a JSON error for a missing service](example_0.gif)

## Try it

Build the executable from the repository root so you can inspect its exit status
and stderr directly:

```sh
demo_dir=$(mktemp -d)
go build -o "$demo_dir/inventory" ./examples/inventory
"$demo_dir/inventory" list
"$demo_dir/inventory" list --output yaml
"$demo_dir/inventory" list --output json -vvv > "$demo_dir/services.json" 2> "$demo_dir/diagnostics.log"
"$demo_dir/inventory" show unknown --error-format json
```

The table renders documentation links when the terminal supports them. The
JSON export stays valid even at trace verbosity because results go to stdout
and diagnostics go to stderr. The missing service writes
`{"code":"service_not_found","message":"Service \"unknown\" does not exist."}`
to stderr and exits unsuccessfully.

## In the code

- [`flags.render`](main.go) selects the result renderer and emits a diagnostic
  without mixing it into the result stream.
- [`run`](main.go) returns a `CodedError` for a failed lookup. The entry point
  chooses its format with `RenderError` and sets the process exit status.
