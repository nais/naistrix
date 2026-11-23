# Render structured errors

The `fail` command returns a `CodedError` with a stable code and a public
message. The CLI entry point renders it on stderr and exits with status 1.
The example defaults to JSON. From the repository root:

```bash
go build -o /tmp/naistrix-errors ./examples/errors
/tmp/naistrix-errors fail
```

```json
{"code":"permission_denied","message":"You cannot update this resource."}
```

Use the application's global `--error-format` flag to try YAML or plain text:

```bash
/tmp/naistrix-errors --error-format=yaml fail
/tmp/naistrix-errors --error-format=text fail
```

The flag belongs to this example, not to Naistrix. Other applications can keep
printing errors as plain text without changing anything.
