# Render data as JSON

Applications can render data encoded as JSON.

Results go to stdout and status messages go to stderr by default, with no writer
options needed. Even at trace verbosity, stdout contains only the JSON result:

```bash
go run . show -vvv > users.json 2> diagnostics.log
```
