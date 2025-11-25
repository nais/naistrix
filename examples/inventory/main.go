package main

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/nais/naistrix"
	"github.com/nais/naistrix/output"
)

type service struct {
	Name string      `json:"name" yaml:"name"`
	Team string      `json:"team" yaml:"team"`
	Docs output.Link `json:"docs" yaml:"docs" heading:"Documentation"`
}

var catalog = []service{
	{Name: "gateway", Team: "platform", Docs: output.NewLink("Gateway docs", "https://example.com/gateway")},
	{Name: "catalog", Team: "apps", Docs: output.NewLink("Catalog docs", "https://example.com/catalog")},
}

type flags struct {
	Output      string `short:"o" usage:"Result format: table, json, or yaml."`
	ErrorFormat string `name:"error-format" usage:"Error format: text, json, or yaml."`
}

func (f *flags) validate(context.Context) error {
	if !slices.Contains([]string{"table", "json", "yaml"}, f.Output) {
		return naistrix.Errorf("--output must be table, json, or yaml")
	}
	if !slices.Contains([]string{"text", "json", "yaml"}, f.ErrorFormat) {
		return naistrix.Errorf("--error-format must be text, json, or yaml")
	}
	return nil
}

func (f *flags) render(out *naistrix.OutputWriter, services []service) error {
	out.Debugf("Loaded %d services from the local catalog\n", len(services))
	switch f.Output {
	case "json":
		return out.JSON(output.JSONWithPrettyOutput()).Render(services)
	case "yaml":
		return out.YAML().Render(services)
	default:
		return out.Table().Render(services)
	}
}

func run(args []string, f *flags) error {
	app, _, err := naistrix.NewApplication("inventory", "Inspect a local service catalog.", "v0.0.0")
	if err != nil {
		return err
	}
	if err := app.AddGlobalFlags(f); err != nil {
		return err
	}
	if err := app.AddCommand(&naistrix.Command{
		Name:              "list",
		Title:             "List the services in the catalog.",
		ValidateFlagsFunc: f.validate,
		RunFunc: func(_ context.Context, _ *naistrix.Arguments, out *naistrix.OutputWriter) error {
			return f.render(out, catalog)
		},
	}, &naistrix.Command{
		Name:              "show",
		Title:             "Look up a service by name.",
		Args:              []naistrix.Argument{{Name: "service"}},
		ValidateFlagsFunc: f.validate,
		RunFunc: func(_ context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
			name := args.Get("service")
			for _, entry := range catalog {
				if entry.Name == name {
					return f.render(out, []service{entry})
				}
			}
			return naistrix.NewCodedError("service_not_found", fmt.Sprintf("Service %q does not exist.", name), nil)
		},
	}); err != nil {
		return err
	}
	return app.Run(naistrix.RunWithArgs(args))
}

func main() {
	f := &flags{Output: "table", ErrorFormat: "text"}
	if err := run(os.Args[1:], f); err != nil {
		format := naistrix.ErrorFormat(f.ErrorFormat)
		if format != naistrix.ErrorFormatJSON && format != naistrix.ErrorFormatYAML {
			format = naistrix.ErrorFormatText
		}
		if renderErr := naistrix.RenderError(os.Stderr, err, format); renderErr != nil {
			_, _ = fmt.Fprintln(os.Stderr, renderErr)
		}
		os.Exit(1)
	}
}
