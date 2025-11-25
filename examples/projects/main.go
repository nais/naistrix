package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nais/naistrix"
)

var projectsByTeam = map[string][]string{
	"platform": {"gateway", "metrics"},
	"apps":     {"checkout", "catalog"},
}

func projectCommands() *naistrix.Command {
	flags := &struct {
		Team string `short:"t" usage:"Team that owns the projects."`
	}{Team: "platform"}
	listFlags := &struct {
		Offline bool `usage:"Skip demo authentication and read the local catalog."`
	}{}
	return &naistrix.Command{
		Name:        "project",
		Title:       "Browse projects owned by a team.",
		Group:       "Projects",
		StickyFlags: flags,
		PreRunHooks: []naistrix.PreRunHook{{
			Name: "select-team",
			RunFunc: func(_ context.Context, out *naistrix.OutputWriter) error {
				if _, ok := projectsByTeam[flags.Team]; !ok {
					return naistrix.Errorf("unknown team %q (choose platform or apps)", flags.Team)
				}
				out.Verbosef("Using team %s\n", flags.Team)
				return nil
			},
		}},
		SubCommands: []*naistrix.Command{{
			Name:            "list",
			Title:           "List projects for the selected team.",
			Aliases:         []string{"ls"},
			TopLevelAliases: []string{"list"},
			Flags:           listFlags,
			SkipPreRunHooksFunc: func() []string {
				if listFlags.Offline {
					return []string{"auth"}
				}
				return nil
			},
			Examples: []naistrix.CommandExample{{
				Description: "List projects owned by the apps team.",
				Command:     "--team apps",
			}},
			RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
				for _, project := range projectsByTeam[flags.Team] {
					out.Println(project)
				}
				return nil
			},
		}},
	}
}

func run(args []string) error {
	flags := &struct {
		Profile string `usage:"Demo account profile."`
	}{Profile: "demo"}
	app, _, err := naistrix.NewApplication("projects", "Browse a local catalog of team projects.", "v0.0.0",
		naistrix.ApplicationWithDefaultsCommandName("config"),
		naistrix.ApplicationWithDefaultsDocumentation(naistrix.DefaultDefinition{
			Key:         "team",
			Title:       "Default project team.",
			Description: "Set the team used by project commands unless <info>--team</info> is supplied.",
		}),
		naistrix.ApplicationWithDefaultsCommandSkipPreRunHooks("auth"),
		naistrix.ApplicationWithPreRunHooks(naistrix.PreRunHook{
			Name: "auth",
			RunFunc: func(_ context.Context, out *naistrix.OutputWriter) error {
				if flags.Profile != "demo" {
					return naistrix.Errorf("this local catalog only accepts the demo profile")
				}
				out.Verboseln("Using the demo profile")
				return nil
			},
		}),
	)
	if err != nil {
		return err
	}
	if err := app.AddGlobalFlags(flags); err != nil {
		return err
	}
	if err := app.AddCommand(projectCommands(), &naistrix.Command{
		Name:            "show-projects",
		Title:           "List projects using the old command name.",
		SkipPreRunHooks: []string{"auth"},
		Deprecated:      naistrix.DeprecatedWithReplacement([]string{"project", "list"}),
	}); err != nil {
		return err
	}
	if len(args) > 0 && args[0] == "--generate-docs" {
		if len(args) != 2 || args[1] == "" {
			return fmt.Errorf("usage: projects --generate-docs DIRECTORY")
		}
		return app.GenerateDocs(naistrix.GenerateDocsWithTargetDir(args[1]))
	}
	return app.Run(naistrix.RunWithArgs(args))
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
