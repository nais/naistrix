package naistrix_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nais/naistrix"
	"github.com/spf13/cobra"
)

func TestPositionalCompletionValidatesSynchronizedFlags(t *testing.T) {
	for _, mode := range []string{"custom", "file extensions", "no handler"} {
		for _, failure := range []string{"none", "parent", "child", "configuration"} {
			t.Run(mode+"/"+failure, func(t *testing.T) {
				app, stdout, stderr := newCompletionApplication(t)
				global := &struct{ Region string }{}
				parentFlags := &struct{ Project string }{}
				childFlags := &struct{ Team string }{}
				if err := app.AddGlobalFlags(global); err != nil {
					t.Fatal(err)
				}
				config := filepath.Join(t.TempDir(), "config.yaml")
				contents := "region: eu\nproject: platform\nteam: Dev\n"
				if failure == "configuration" {
					contents = "team: [\n"
				}
				if err := os.WriteFile(config, []byte(contents), 0o600); err != nil {
					t.Fatal(err)
				}
				var events []string
				assertSynchronized := func() {
					t.Helper()
					if global.Region != "eu" || parentFlags.Project != "platform" || childFlags.Team != "Dev" {
						t.Errorf("unsynchronized flags: region=%q, project=%q, team=%q", global.Region, parentFlags.Project, childFlags.Team)
					}
				}
				validator := func(name string) naistrix.ValidateFlagsFunc {
					return func(context.Context) error {
						assertSynchronized()
						events = append(events, name)
						if failure == name {
							return errors.New(name + " validation failed")
						}
						return nil
					}
				}
				child := &naistrix.Command{
					Name: "child", Title: "Child", Flags: childFlags,
					ValidateFlagsFunc: validator("child"), RunFunc: completionMustNotRun(t),
					ValidateArgsFunc: func(context.Context, *naistrix.Arguments) error {
						t.Error("positional completion ran argument validation")
						return nil
					},
				}
				switch mode {
				case "custom":
					child.AutoCompleteFunc = func(context.Context, *naistrix.Arguments, string) ([]string, string) {
						assertSynchronized()
						events = append(events, "completion")
						return []string{"custom"}, ""
					}
				case "file extensions":
					child.AutoCompleteExtensions = []string{"yaml", "json"}
				}
				if err := app.AddCommand(&naistrix.Command{
					Name: "parent", Title: "Parent", StickyFlags: parentFlags,
					ValidateFlagsFunc: validator("parent"), SubCommands: []*naistrix.Command{child},
					ValidateArgsFunc: func(context.Context, *naistrix.Arguments) error {
						t.Error("positional completion ran ancestor argument validation")
						return nil
					},
				}); err != nil {
					t.Fatal(err)
				}
				got, directive := requestCompletion(t, app, stdout, "parent", "child", "--config", config, "")
				wantEvents := []string{"parent", "child"}
				if failure != "none" {
					switch failure {
					case "parent":
						wantEvents = []string{"parent"}
					case "configuration":
						wantEvents = nil
					}
					if directive != cobra.ShellCompDirectiveNoFileComp || !strings.Contains(strings.Join(got, " "), "Completion failed:") || !strings.Contains(stderr.String(), "Completion failed:") {
						t.Errorf("expected completion diagnostic, got %v, directive %v, stderr %q", got, directive, stderr.String())
					}
				} else {
					switch mode {
					case "custom":
						wantEvents = append(wantEvents, "completion")
						if !slices.Equal(got, []string{"custom"}) || directive != cobra.ShellCompDirectiveNoFileComp {
							t.Errorf("got %v, directive %v", got, directive)
						}
					case "file extensions":
						if !slices.Contains(got, "yaml") || !slices.Contains(got, "json") || directive != cobra.ShellCompDirectiveFilterFileExt {
							t.Errorf("got %v, directive %v", got, directive)
						}
					case "no handler":
						if len(got) != 0 || directive != cobra.ShellCompDirectiveNoFileComp {
							t.Errorf("got %v, directive %v", got, directive)
						}
					}
				}
				if !slices.Equal(events, wantEvents) {
					t.Errorf("events %v, want %v", events, wantEvents)
				}
			})
		}
	}
}

type synchronizedCompletionValue string

type synchronizedCompletionFlags struct {
	Value    synchronizedCompletionValue
	Setting  string
	complete func(context.Context, *naistrix.Arguments, string) ([]string, string)
}

func (*synchronizedCompletionValue) AutoComplete(ctx context.Context, args *naistrix.Arguments, toComplete string, flags any) ([]string, string) {
	return flags.(*synchronizedCompletionFlags).complete(ctx, args, toComplete)
}

func TestFlagCompletionSynchronizesWithoutValidation(t *testing.T) {
	for _, scope := range []string{"command", "sticky", "global"} {
		for _, source := range []string{"CLI", "configuration", "environment", "bad configuration"} {
			t.Run(scope+"/"+source, func(t *testing.T) {
				app, stdout, stderr := newCompletionApplication(t)
				flags := &synchronizedCompletionFlags{}
				global := &struct{ Region string }{}
				parentFlags := &struct{ Project string }{}
				childFlags := &struct{ Mode string }{}
				callbacks := 0
				flags.complete = func(context.Context, *naistrix.Arguments, string) ([]string, string) {
					callbacks++
					if flags.Setting != "Dev" {
						t.Errorf("completion saw unsynchronized setting %q", flags.Setting)
					}
					if scope != "global" && global.Region != "eu" {
						t.Errorf("unsynchronized global region %q", global.Region)
					}
					if scope != "sticky" && parentFlags.Project != "platform" {
						t.Errorf("unsynchronized parent project %q", parentFlags.Project)
					}
					if scope != "command" && childFlags.Mode != "preview" {
						t.Errorf("unsynchronized target mode %q", childFlags.Mode)
					}
					return []string{"suggestion"}, ""
				}
				mustNotValidate := func(context.Context) error {
					t.Error("flag-value completion ran command-wide validation")
					return errors.New("required flag missing")
				}
				child := &naistrix.Command{
					Name: "child", Title: "Child", Flags: childFlags, RunFunc: completionMustNotRun(t),
					ValidateFlagsFunc: mustNotValidate,
					ValidateArgsFunc: func(context.Context, *naistrix.Arguments) error {
						t.Error("flag-value completion ran argument validation")
						return nil
					},
				}
				parent := &naistrix.Command{
					Name: "parent", Title: "Parent", StickyFlags: parentFlags,
					ValidateFlagsFunc: mustNotValidate, SubCommands: []*naistrix.Command{child},
				}
				var globalFlags any = global
				switch scope {
				case "command":
					child.Flags = flags
				case "sticky":
					parent.StickyFlags = flags
				case "global":
					globalFlags = flags
				}
				if err := app.AddGlobalFlags(globalFlags); err != nil {
					t.Fatal(err)
				}
				if err := app.AddCommand(parent); err != nil {
					t.Fatal(err)
				}
				args := []string{"parent", "child"}
				switch source {
				case "CLI":
					args = append(args, "--setting", "Dev")
					if scope != "global" {
						args = append(args, "--region", "eu")
					}
					if scope != "sticky" {
						args = append(args, "--project", "platform")
					}
					if scope != "command" {
						args = append(args, "--mode", "preview")
					}
				case "environment":
					for key, value := range map[string]string{"SETTING": "Dev", "REGION": "eu", "PROJECT": "platform", "MODE": "preview"} {
						t.Setenv("COMPLETIONTEST_"+key, value)
					}
				default:
					contents := "setting: Dev\nregion: eu\nproject: platform\nmode: preview\n"
					if source == "bad configuration" {
						contents = "setting: [\n"
					}
					path := filepath.Join(t.TempDir(), "config.yaml")
					if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
						t.Fatal(err)
					}
					args = append(args, "--config", path)
				}
				got, directive := requestCompletion(t, app, stdout, append(args, "--value", "")...)
				if source == "bad configuration" {
					if callbacks != 0 || !strings.Contains(strings.Join(got, " "), "Completion failed:") || !strings.Contains(stderr.String(), "Completion failed:") {
						t.Errorf("expected synchronization error before callback; calls=%d, suggestions=%v, stderr=%q", callbacks, got, stderr.String())
					}
				} else if callbacks != 1 || !slices.Equal(got, []string{"suggestion"}) {
					t.Errorf("callbacks=%d, suggestions=%v", callbacks, got)
				}
				if directive != cobra.ShellCompDirectiveNoFileComp {
					t.Errorf("unexpected directive %v", directive)
				}
			})
		}
	}
}

type synchronizedFileValue string

func (v *synchronizedFileValue) FileExtensions() []string {
	// Derive the extension from the synchronized value to detect callbacks run before configuration is loaded.
	return []string{string(*v)}
}

func TestFileFlagCompletionSynchronizesWithoutValidation(t *testing.T) {
	for _, scope := range []string{"command", "sticky", "global"} {
		t.Run(scope, func(t *testing.T) {
			app, stdout, _ := newCompletionApplication(t)
			flags := &struct{ File synchronizedFileValue }{File: "json"}
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte("file: yaml\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			mustNotValidate := func(context.Context) error {
				t.Error("file flag completion ran command-wide validation")
				return errors.New("required flag missing")
			}
			child := &naistrix.Command{
				Name: "child", Title: "Child", RunFunc: completionMustNotRun(t), ValidateFlagsFunc: mustNotValidate,
				AutoCompleteFunc: func(context.Context, *naistrix.Arguments, string) ([]string, string) {
					t.Error("file flag completion ran positional callback")
					return nil, ""
				},
			}
			parent := &naistrix.Command{
				Name: "parent", Title: "Parent", SubCommands: []*naistrix.Command{child}, ValidateFlagsFunc: mustNotValidate,
			}
			switch scope {
			case "command":
				child.Flags = flags
			case "sticky":
				parent.StickyFlags = flags
			case "global":
				if err := app.AddGlobalFlags(flags); err != nil {
					t.Fatal(err)
				}
			}
			if err := app.AddCommand(parent); err != nil {
				t.Fatal(err)
			}
			for range 2 {
				got, directive := requestCompletion(t, app, stdout, "parent", "child", "--config", path, "--file", "")
				if len(got) != 2 || got[0] != "yaml" || directive != cobra.ShellCompDirectiveFilterFileExt {
					t.Fatalf("expected synchronized extensions and one help message, got %v, directive %v", got, directive)
				}
			}
		})
	}
}
