package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nais/naistrix"
	"github.com/nais/naistrix/input"
)

type CreateFlags struct {
	Team     string `name:"team" usage:"Team that owns the application."`
	Replicas int    `name:"replicas" usage:"Number of application replicas."`
}

func createCommand() *naistrix.Command {
	flags := &CreateFlags{Replicas: 2}
	return &naistrix.Command{
		Name: "create", Title: "Create an application.", Flags: flags,
		Args: []naistrix.Argument{{Name: "name"}},
		WizardFunc: func(ctx context.Context, _ *naistrix.OutputWriter) ([]string, error) {
			name, err := input.Input(ctx, "Application name")
			if err != nil {
				return nil, err
			}
			// Defaults are already loaded. Keep the configured team when present.
			if strings.TrimSpace(flags.Team) == "" {
				team, err := input.Input(ctx, "Team that owns the application")
				if err != nil {
					return nil, err
				}
				flags.Team = strings.TrimSpace(team)
			}
			customize, err := input.Confirm(ctx, fmt.Sprintf("Change the replica count from %d?", flags.Replicas))
			if err != nil {
				return nil, err
			}
			if customize {
				replicas, err := input.Select(ctx, "Number of replicas", []int{1, 2, 3})
				if err != nil {
					return nil, err
				}
				flags.Replicas = replicas
			}
			return []string{strings.TrimSpace(name)}, nil
		},
		ValidateFlagsFunc: func(context.Context) error {
			if strings.TrimSpace(flags.Team) == "" {
				return naistrix.Errorf("--team is required")
			}
			if flags.Replicas < 1 {
				return naistrix.Errorf("--replicas must be at least 1")
			}
			return nil
		},
		ValidateArgsFunc: func(_ context.Context, args *naistrix.Arguments) error {
			if strings.TrimSpace(args.Get("name")) == "" {
				return naistrix.Errorf("application name must not be empty")
			}
			return nil
		},
		RunFunc: func(_ context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
			out.Printf("Would create %s for team %s with %d replicas.\n", args.Get("name"), flags.Team, flags.Replicas)
			return nil
		},
	}
}

func main() {
	app, _, err := naistrix.NewApplication("wizard-example", "Example application with a command wizard.", "v0.0.0")
	if err == nil {
		err = app.AddCommand(createCommand())
	}
	if err == nil {
		err = app.Run()
	}
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
