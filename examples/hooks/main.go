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
		Name: "auth", Title: "Authentication commands",
		// All authentication subcommands manage their own authentication flow.
		SkipPreRunHooks: []string{"auth"},
		SubCommands: []*naistrix.Command{{
			Name: "login", Title: "Run the demo login flow", TopLevelAliases: []string{"login"},
			RunFunc: func(ctx context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
				return authenticate(ctx, out)
			},
		}},
	}); err != nil {
		return err
	}
	return app.Run()
}
