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
		fmt.Printf("error when creating application: %v\n", err)
		os.Exit(1)
	}

	err = app.AddCommand(&naistrix.Command{
		Name:  "transform",
		Title: "Transform all the words",
		Args: []naistrix.Argument{
			{Name: "func"},
			{Name: "word", Repeatable: true},
		},
		ValidateFunc: func(ctx context.Context, args *naistrix.Arguments) error {
			switch cb := args.Get("func"); cb {
			case "upper", "lower":
				return nil
			default:
				return naistrix.Errorf(`only "upper" or "lower" is allowed for the "func" argument, got: %q`, cb)
			}
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
		fmt.Printf("error when adding command: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		fmt.Printf("error when running application: %v\n", err)
		os.Exit(1)
	}
}
