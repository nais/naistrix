package main

import (
	"context"
	"log"
	"strings"

	"github.com/nais/naistrix"
)

type GreetFlags struct {
	Loud bool `name:"loud" short:"l" usage:"Print in uppercase"`
}

func main() {
	app, _, err := naistrix.NewApplication(
		"example",
		"Example CLI for documentation generation",
		"0.1.0",
	)
	if err != nil {
		log.Fatalf("failed to create app: %v", err)
	}

	greetFlags := &GreetFlags{}
	if err := app.AddCommand(&naistrix.Command{
		Name:        "greet",
		Title:       "Greet someone",
		Description: "Prints a greeting message.",
		Args:        []naistrix.Argument{{Name: "name"}},
		Flags:       greetFlags,
		RunFunc: func(_ context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
			name := args.Get("name")
			msg := "Hello, " + name + "!"
			if greetFlags.Loud {
				msg = strings.ToUpper(msg)
			}
			out.Println(msg)
			return nil
		},
	}); err != nil {
		log.Fatalf("failed to add command: %v", err)
	}

	if err := app.GenerateDocs(); err != nil {
		log.Fatalf("failed to generate docs: %v", err)
	}
}
