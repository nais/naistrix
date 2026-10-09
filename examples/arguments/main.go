package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nais/naistrix"
)

func main() {
	app, _, err := naistrix.NewApplication(
		"example",
		"Example application with command arguments",
		"v0.0.0",
	)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error when creating application: %v\n", err)
		os.Exit(1)
	}

	err = app.AddCommand(&naistrix.Command{
		Name:  "transform",
		Title: "Transform all the words",
		Args: []naistrix.Argument{
			{Name: "func", Prompt: "How should the words be transformed?", Choices: []string{"upper", "lower"}, ChoicesCaseInsensitive: true},
			{Name: "word", Repeatable: true, Prompt: "Enter a word to transform"},
		},
		RunFunc: func(ctx context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
			var t func(string) string
			if args.Get("func") == "upper" {
				t = strings.ToUpper
			} else {
				t = strings.ToLower
			}

			out.Printf("Words: ")
			w := args.GetRepeatable("word")
			words := make([]any, len(w))
			for i, word := range w {
				words[i] = t(word)
			}
			out.Println(words...)
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
