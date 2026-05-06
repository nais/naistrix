package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nais/naistrix"
)

func main() {
	app, _, err := naistrix.NewApplication(
		"example",
		"Example application with a renamed defaults command",
		"v0.0.0",
		naistrix.ApplicationWithDefaultsCommandName("config"),
	)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error when creating application: %v\n", err)
		os.Exit(1)
	}

	err = app.AddCommand(&naistrix.Command{
		Name:  "greet",
		Title: "Greet the user",
		Args:  []naistrix.Argument{{Name: "user_name"}},
		RunFunc: func(_ context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
			out.Println("Hello, " + args.Get("user_name") + "!")
			return nil
		},
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error when adding command: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error when running application: %v\n", err)
		os.Exit(1)
	}
}
