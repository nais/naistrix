# Naistrix examples

Each application runs from the repository root and uses local data. The READMEs
include terminal recordings and link to the code behind each workflow.

| Application | Start here if you want to… |
| --- | --- |
| [`projects`](projects/) | Organize commands, load team defaults, and migrate a legacy command. |
| [`deploy`](deploy/) | Guide a user through a release, replay it in CI, or complete a release name. |
| [`inventory`](inventory/) | Make the same CLI useful in a terminal and a pipeline. |

## Quickstart

Save this as `main.go` in a Go module that depends on `github.com/nais/naistrix`,
then run `go run . greet World`.

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nais/naistrix"
)

func run() error {
	app, _, err := naistrix.NewApplication("hello", "Print a greeting.", "v0.0.0")
	if err != nil {
		return err
	}
	if err := app.AddCommand(&naistrix.Command{
		Name:  "greet",
		Title: "Greet someone by name.",
		Args:  []naistrix.Argument{{Name: "name"}},
		RunFunc: func(_ context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
			out.Printf("Hello, %s!\n", args.Get("name"))
			return nil
		},
	}); err != nil {
		return err
	}
	return app.Run()
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

See the [recording task](../mise/tasks/examples/README.md) to regenerate the previews.
