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
		"Example application with a few deprecated commands",
		"v0.0.0",
	)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error when creating application: %v\n", err)
		os.Exit(1)
	}

	err = app.AddCommand(
		&naistrix.Command{
			Name:       "command-v1",
			Title:      "This is the first version of the command",
			Deprecated: naistrix.DeprecatedWithReplacement([]string{"command-v2"}),
			RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
				out.Println("do some stuff")
				return nil
			},
		},
		&naistrix.Command{
			Name:  "command-v2",
			Title: "This is the second version of the command",
			Deprecated: naistrix.DeprecatedWithReplacementFunc(func(_ context.Context, args *naistrix.Arguments) []string {
				return []string{"command-v3", "value-for-arg"}
			}),
			RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
				out.Println("do some stuff")
				return nil
			},
		},
		&naistrix.Command{
			Name:  "command-v3",
			Title: "This is the latest and greatest version of the command",
			Args:  []naistrix.Argument{{Name: "bar"}},
			RunFunc: func(_ context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
				out.Println("bar:", args.Get("bar"))
				return nil
			},
		},
		&naistrix.Command{
			Name:       "removed-command",
			Title:      "This command has been removed without a replacement",
			Deprecated: naistrix.DeprecatedWithoutReplacement(),
			RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
				out.Println("this should never run")
				return nil
			},
		},
	)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error when adding command: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error when running application: %v\n", err)
		os.Exit(1)
	}
}
