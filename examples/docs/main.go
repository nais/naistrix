package main

import (
	"context"
	"log"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
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
		naistrix.ApplicationWithDefaultsDocumentation(naistrix.DefaultDefinition{
			Key:   "loud",
			Title: "Uppercase greetings.",
			Description: strings.TrimSpace(heredoc.Doc(`
				Print greetings in uppercase when <info>--loud</info> is omitted.

				An explicit <info>--loud=false</info> flag overrides this setting.
			`)),
		}),
	)
	if err != nil {
		log.Fatalf("failed to create app: %v", err)
	}

	greetFlags := &GreetFlags{}
	if err := app.AddCommand(&naistrix.Command{
		Name:  "greet",
		Title: "Greet someone",
		Description: strings.TrimSpace(heredoc.Doc(`
			Print a greeting for <info><name></info>.

			Use <info>--loud</info> to print the greeting in uppercase.
		`)),
		Args:  []naistrix.Argument{{Name: "name"}},
		Flags: greetFlags,
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
