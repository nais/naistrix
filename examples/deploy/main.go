package main

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/nais/naistrix"
	"github.com/nais/naistrix/input"
	"gopkg.in/yaml.v3"
)

var (
	environments = []string{"staging", "production"}
	releases     = map[string][]string{
		"staging":    {"gateway-v2.1", "catalog-v1.4"},
		"production": {"gateway-v2.0", "catalog-v1.3"},
	}
)

func planCommand() *naistrix.Command {
	flags := &struct {
		Environment string `short:"e" usage:"Environment for the release."`
	}{}
	return &naistrix.Command{
		Name:  "plan",
		Title: "Preview a new release without deploying it.",
		Flags: flags,
		Args:  []naistrix.Argument{{Name: "service"}},
		WizardFunc: func(ctx context.Context, _ *naistrix.OutputWriter) ([]string, error) {
			service, err := input.Input(ctx, "Service name")
			if err != nil {
				return nil, err
			}
			if flags.Environment == "" {
				flags.Environment, err = input.Select(ctx, "Target environment", environments)
				if err != nil {
					return nil, err
				}
			}
			return []string{strings.TrimSpace(service)}, nil
		},
		ValidateFlagsFunc: func(context.Context) error {
			if !slices.Contains(environments, flags.Environment) {
				return naistrix.Errorf("--environment must be staging or production")
			}
			return nil
		},
		ValidateArgsFunc: func(_ context.Context, args *naistrix.Arguments) error {
			if strings.TrimSpace(args.Get("service")) == "" {
				return naistrix.Errorf("service name must not be empty")
			}
			return nil
		},
		RunFunc: func(_ context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
			out.Printf("Plan: release %s to %s\n", args.Get("service"), flags.Environment)
			return nil
		},
	}
}

func promoteCommand() *naistrix.Command {
	flags := &struct {
		Set    naistrix.StringArray `usage:"Repeatable KEY=VALUE override. Commas are preserved."`
		Notify *bool                `usage:"Override whether to notify the team."`
	}{}
	return &naistrix.Command{
		Name:  "promote",
		Title: "Choose a release from an environment.",
		Flags: flags,
		Args: []naistrix.Argument{{
			Name:                   "environment",
			Choices:                environments,
			ChoicesCaseInsensitive: true,
		}, {
			Name: "release",
			PromptFunc: func(_ context.Context, args *naistrix.Arguments) (string, error) {
				return fmt.Sprintf("Choose a release from %s", args.Get("environment")), nil
			},
			ChoicesFunc: func(_ context.Context, args *naistrix.Arguments) ([]string, error) {
				return releases[args.Get("environment")], nil
			},
		}},
		ValidateFlagsFunc: func(context.Context) error {
			for _, override := range flags.Set {
				key, _, ok := strings.Cut(override, "=")
				if !ok || key == "" {
					return naistrix.Errorf("--set expects KEY=VALUE, got %q", override)
				}
			}
			return nil
		},
		RunFunc: func(_ context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
			out.Printf("Would promote %s from %s\n", args.Get("release"), args.Get("environment"))
			for _, override := range flags.Set {
				out.Printf("  %s\n", override)
			}
			if flags.Notify != nil {
				out.Printf("  notify=%t\n", *flags.Notify)
			}
			return nil
		},
	}
}

func checkCommand() *naistrix.Command {
	return &naistrix.Command{
		Name:  "check",
		Title: "Check the service name in each local manifest.",
		Args: []naistrix.Argument{{
			Name:                   "manifest",
			Repeatable:             true,
			AutoCompleteExtensions: []string{"yaml", "yml", "json"},
		}},
		RunFunc: func(_ context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
			for _, path := range args.GetRepeatable("manifest") {
				// #nosec G304 -- This local CLI reads the manifest path explicitly supplied by the caller.
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				var manifest struct {
					Service string `yaml:"service"`
				}
				if err := yaml.Unmarshal(data, &manifest); err != nil {
					return fmt.Errorf("read %s: %w", path, err)
				}
				if strings.TrimSpace(manifest.Service) == "" {
					return naistrix.Errorf("%s: missing service name", path)
				}
				out.Printf("%s: %s\n", path, manifest.Service)
			}
			return nil
		},
	}
}

func run(args []string) error {
	app, _, err := naistrix.NewApplication("deploy", "Plan releases from a local catalog.", "v0.0.0")
	if err != nil {
		return err
	}
	if err := app.AddCommand(planCommand(), promoteCommand(), checkCommand()); err != nil {
		return err
	}
	return app.Run(naistrix.RunWithArgs(args))
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
