package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nais/naistrix"
	"github.com/nais/naistrix/input"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	flags := &struct {
		Token string `usage:"Demo access token (any non-empty value)."`
	}{}
	// This only simulates authentication. A real application would check stored
	// credentials, refresh tokens, and start its login flow when necessary.
	authenticate := func(ctx context.Context, out *naistrix.OutputWriter) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if flags.Token == "" {
			token, err := input.Input("Demo access token")
			if err != nil {
				return err
			}
			flags.Token = strings.TrimSpace(token)
		}
		if flags.Token == "" {
			return naistrix.Errorf("an access token is required")
		}
		out.Println("Authenticated (demo only)")
		return nil
	}
	app, _, err := naistrix.NewApplication("example", "Lifecycle hook example", "v0.0.0",
		naistrix.ApplicationWithDefaultsCommandSkipPreRunHooks("auth"),
		naistrix.ApplicationWithPreRunHooks(naistrix.PreRunHook{
			Name: "auth",
			RunFunc: func(ctx context.Context, out *naistrix.OutputWriter) error {
				return authenticate(ctx, out)
			},
		}),
	)
	if err != nil {
		return err
	}
	if err := app.AddGlobalFlags(flags); err != nil {
		return err
	}
	projectFlags := &struct {
		Project string `usage:"Demo project to prepare before running a project command."`
	}{Project: "demo"}
	inspectFlags := &struct {
		Offline bool `usage:"Inspect local data without authentication."`
	}{}
	if err := app.AddCommand(&naistrix.Command{
		Name: "members", Title: "Select a member after authentication",
		Args: []naistrix.Argument{{
			Name: "member", Prompt: "Select a member",
			ChoicesFunc: func(context.Context, *naistrix.Arguments) ([]string, error) {
				// An authenticated API call could supply these choices.
				return []string{"Alice", "Bob"}, nil
			},
		}},
		RunFunc: func(_ context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
			out.Printf("Selected %s\n", args.Get("member"))
			return nil
		},
	}, &naistrix.Command{
		Name: "inspect", Title: "Inspect data, optionally offline", Flags: inspectFlags,
		SkipPreRunHooksFunc: func() []string {
			if inspectFlags.Offline {
				return []string{"auth"}
			}
			return nil
		},
		RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
			if inspectFlags.Offline {
				out.Println("Inspecting local data without authentication")
			} else {
				out.Println("Inspecting data after authentication")
			}
			return nil
		},
	}, &naistrix.Command{
		Name: "auth", Title: "Authentication commands",
		// All authentication subcommands manage their own authentication flow.
		SkipPreRunHooks: []string{"auth"},
		SubCommands: []*naistrix.Command{{
			Name: "login", Title: "Run the demo login flow", TopLevelAliases: []string{"login"},
			RunFunc: func(ctx context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
				return authenticate(ctx, out)
			},
		}},
	}, &naistrix.Command{
		Name: "projects", Title: "Commands with inherited project setup", StickyFlags: projectFlags,
		PreRunHooks: []naistrix.PreRunHook{{
			Name: "project",
			RunFunc: func(_ context.Context, out *naistrix.OutputWriter) error {
				out.Printf("Prepared project %s (demo only)\n", projectFlags.Project)
				return nil
			},
		}},
		SubCommands: []*naistrix.Command{
			{
				Name: "show", Title: "Run application authentication, then inherited project setup", TopLevelAliases: []string{"show-project"},
				RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
					out.Printf("Selected project %s\n", projectFlags.Project)
					return nil
				},
			},
			{
				Name: "public", Title: "Skip authentication and inherited project setup", SkipPreRunHooks: []string{"auth", "project"},
				RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
					out.Println("No authentication or project setup needed")
					return nil
				},
			},
		},
	}); err != nil {
		return err
	}
	return app.Run()
}
