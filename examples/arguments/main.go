package main

import (
	"context"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/nais/naistrix"
)

var members = map[string][]string{
	"platform": {"Alice", "Bob"},
	"apps":     {"Carol", "Dave"},
}

func getTeamMembers(team string) ([]string, error) {
	if team == "" {
		return nil, naistrix.Errorf("specify a team using --team")
	}
	m, ok := members[team]
	if !ok {
		return nil, naistrix.Errorf("team %q not found, available teams: %s", team, strings.Join(slices.Collect(maps.Keys(members)), ", "))
	}
	return slices.Clone(m), nil
}

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

	flags := &struct {
		Team string `usage:"Team to select a member from."`
	}{}
	err = app.AddCommand(&naistrix.Command{
		Name:  "transform",
		Title: "Transform all the words",
		Args: []naistrix.Argument{{
			Name:                   "transformation",
			Prompt:                 "How should the words be transformed?",
			Choices:                []string{"upper", "lower"},
			ChoicesCaseInsensitive: true,
		}, {
			Name:       "word",
			Repeatable: true,
			Prompt:     "Enter a word to transform",
		}},
		RunFunc: func(ctx context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
			var t func(string) string
			if args.Get("transformation") == "upper" {
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
	}, &naistrix.Command{
		Name:  "delete-member",
		Title: "Preview deleting a member from a team",
		Flags: flags,
		Args: []naistrix.Argument{{
			Name: "member",
			PromptFunc: func(context.Context, *naistrix.Arguments) (string, error) {
				return fmt.Sprintf("Select a member of the %q team", flags.Team), nil
			},
			ChoicesCaseInsensitive: true,
			ChoicesFunc: func(context.Context, *naistrix.Arguments) ([]string, error) {
				return getTeamMembers(flags.Team)
			},
		}},
		RunFunc: func(_ context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
			out.Printf("Would delete %q from the %q team\n", args.Get("member"), flags.Team)
			return nil
		},
	}, &naistrix.Command{
		Name:  "members",
		Title: "Select members using an earlier team argument",
		Args: []naistrix.Argument{
			{
				Name:                   "team",
				Choices:                []string{"platform", "apps"},
				ChoicesCaseInsensitive: true,
			}, {
				Name:                   "member",
				Repeatable:             true,
				ChoicesCaseInsensitive: true,
				PromptFunc: func(_ context.Context, args *naistrix.Arguments) (string, error) {
					return fmt.Sprintf("Select a member of the %q team", args.Get("team")), nil
				},
				ChoicesFunc: func(_ context.Context, args *naistrix.Arguments) ([]string, error) {
					return getTeamMembers(args.Get("team"))
				},
			},
		},
		RunFunc: func(_ context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
			out.Printf("Team: %q, members: %v\n", args.Get("team"), strings.Join(args.GetRepeatable("member"), ", "))
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
