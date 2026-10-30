package naistrix

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nais/naistrix/input"
)

func newHookApplication(t *testing.T, hooks ...PreRunHook) *Application {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	app, _, err := NewApplication("hooktest", "Hook test", "v1.0.0",
		ApplicationWithWriter(io.Discard), ApplicationWithPreRunHooks(hooks...))
	if err != nil {
		t.Fatal(err)
	}
	app.rootCommand.SetErr(io.Discard)
	return app
}

func TestPreRunHooksBeforeArgumentResolution(t *testing.T) {
	for _, mode := range []string{"supplied", "prompted", "hook failure"} {
		t.Run(mode, func(t *testing.T) {
			var events []string
			blocked := errors.New("authentication failed")
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var expectedOutput *OutputWriter
			app := newHookApplication(t, PreRunHook{
				Name: "auth",
				RunFunc: func(gotCtx context.Context, out *OutputWriter) error {
					events = append(events, "auth")
					if gotCtx != ctx || out != expectedOutput {
						t.Error("hook did not receive the execution context and application output")
					}
					if mode == "hook failure" {
						return blocked
					}
					return nil
				},
			}, PreRunHook{
				Name: "setup",
				RunFunc: func(context.Context, *OutputWriter) error {
					events = append(events, "setup")
					return nil
				},
			})
			expectedOutput = app.Output()
			cmd := &Command{
				Name: "cmd", Title: "Command",
				ValidateFlagsFunc: func(context.Context) error {
					events = append(events, "flags")
					return nil
				},
				Args: []Argument{{
					Name: "member",
					ChoicesFunc: func(context.Context, *Arguments) ([]string, error) {
						events = append(events, "choices")
						return []string{"Alice"}, nil
					},
					PromptFunc: func(context.Context, *Arguments) (string, error) {
						events = append(events, "prompt")
						return "Select member", nil
					},
				}},
				ValidateArgsFunc: func(context.Context, *Arguments) error {
					events = append(events, "arguments")
					return nil
				},
				RunFunc: func(context.Context, *Arguments, *OutputWriter) error {
					events = append(events, "run")
					return nil
				},
			}
			if err := app.AddCommand(cmd); err != nil {
				t.Fatal(err)
			}
			restore := SetChoicePromptForTest(func(string, []string, ...input.SelectOptionFunc) (string, error) {
				events = append(events, "select")
				return "Alice", nil
			})
			defer restore()
			args := []string{"cmd"}
			want := []string{"auth", "setup", "flags", "choices", "prompt", "select", "arguments", "run"}
			switch mode {
			case "supplied":
				args = append(args, "Alice")
				want = []string{"auth", "setup", "flags", "choices", "arguments", "run"}
			case "hook failure":
				want = []string{"auth"}
			}
			err := app.Run(RunWithArgs(args), RunWithContext(ctx))
			if mode == "hook failure" {
				if err != blocked {
					t.Fatalf("expected unchanged hook error, got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(events, want) {
				t.Errorf("events %v, want %v", events, want)
			}
		})
	}
}

func TestPreRunHooksSeeSynchronizedFlags(t *testing.T) {
	for _, source := range []string{"CLI", "configuration", "environment", "default"} {
		t.Run(source, func(t *testing.T) {
			global := &struct{ Region string }{}
			sticky := &struct{ Team string }{}
			local := &struct{ Member string }{}
			calls := 0
			checkFlags := func(context.Context, *OutputWriter) error {
				calls++
				if global.Region != "eu" || sticky.Team != "platform" || local.Member != "Alice" {
					t.Errorf("unsynchronized flags: region=%q, team=%q, member=%q", global.Region, sticky.Team, local.Member)
				}
				return nil
			}
			app := newHookApplication(t, PreRunHook{Name: "auth", RunFunc: checkFlags})
			args := []string{"parent", "child"}
			switch source {
			case "CLI":
				args = append(args, "--region", "eu", "--team", "platform", "--member", "Alice")
			case "configuration":
				path := filepath.Join(t.TempDir(), "config.yaml")
				if err := os.WriteFile(path, []byte("region: eu\nteam: platform\nmember: Alice\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				args = append(args, "--config", path)
			case "environment":
				t.Setenv("HOOKTEST_REGION", "eu")
				t.Setenv("HOOKTEST_TEAM", "platform")
				t.Setenv("HOOKTEST_MEMBER", "Alice")
			case "default":
				global.Region, sticky.Team, local.Member = "eu", "platform", "Alice"
			}
			if err := app.AddGlobalFlags(global); err != nil {
				t.Fatal(err)
			}
			if err := app.AddCommand(&Command{
				Name: "parent", Title: "Parent", StickyFlags: sticky,
				PreRunHooks: []PreRunHook{{Name: "parent", RunFunc: checkFlags}},
				SubCommands: []*Command{{
					Name: "child", Title: "Child", Flags: local, RunFunc: hookNoop,
					PreRunHooks: []PreRunHook{{Name: "child", RunFunc: checkFlags}},
				}},
			}); err != nil {
				t.Fatal(err)
			}
			if err := app.Run(RunWithArgs(args)); err != nil {
				t.Fatal(err)
			}
			if calls != 3 {
				t.Errorf("hooks called %d times, want 3", calls)
			}
		})
	}
}

func TestPreRunHooksBeforeParentValidation(t *testing.T) {
	var events []string
	app := newHookApplication(t, PreRunHook{Name: "auth", RunFunc: func(context.Context, *OutputWriter) error {
		events = append(events, "auth")
		return nil
	}})
	if err := app.AddCommand(&Command{
		Name: "parent", Title: "Parent",
		ValidateFlagsFunc: func(context.Context) error { events = append(events, "parent flags"); return nil },
		Args: []Argument{{Name: "member", ChoicesFunc: func(context.Context, *Arguments) ([]string, error) {
			events = append(events, "parent choices")
			return []string{"Alice"}, nil
		}}},
		SubCommands: []*Command{{Name: "child", Title: "Child", RunFunc: hookNoop}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(RunWithArgs([]string{"parent", "child", "Alice"})); err != nil {
		t.Fatal(err)
	}
	if want := []string{"auth", "parent flags", "parent choices"}; !slices.Equal(events, want) {
		t.Errorf("events %v, want %v", events, want)
	}
}

func TestSkipPreRunHooks(t *testing.T) {
	for _, tt := range []struct {
		args []string
		want []string
	}{
		{args: []string{"protected"}, want: []string{"auth", "setup", "audit"}},
		{args: []string{"public"}, want: []string{"setup", "audit"}},
		{args: []string{"p"}, want: []string{"setup", "audit"}},
		{args: []string{"auth", "login"}, want: []string{"setup"}},
		{args: []string{"login"}, want: []string{"setup"}},
		{args: []string{"auth", "logout"}, want: []string{"setup", "audit"}},
	} {
		t.Run(strings.Join(tt.args, "/"), func(t *testing.T) {
			var events []string
			var hooks []PreRunHook
			for _, name := range []string{"auth", "setup", "audit"} {
				hooks = append(hooks, PreRunHook{Name: name, RunFunc: func(context.Context, *OutputWriter) error {
					events = append(events, name)
					return nil
				}})
			}
			app := newHookApplication(t, hooks...)
			if err := app.AddCommand(
				&Command{Name: "protected", Title: "Protected", RunFunc: hookNoop},
				&Command{Name: "public", Title: "Public", Aliases: []string{"p"}, SkipPreRunHooks: []string{"auth"}, RunFunc: hookNoop},
				&Command{Name: "auth", Title: "Auth", SkipPreRunHooks: []string{"auth"}, SubCommands: []*Command{
					{Name: "login", Title: "Login", TopLevelAliases: []string{"login"}, SkipPreRunHooks: []string{"audit"}, RunFunc: hookNoop},
					{Name: "logout", Title: "Logout", RunFunc: hookNoop},
				}},
			); err != nil {
				t.Fatal(err)
			}
			// Repeated runs must neither retain skip state nor duplicate alias hooks.
			for range 2 {
				events = nil
				if err := app.Run(RunWithArgs(tt.args)); err != nil {
					t.Fatal(err)
				}
				if !slices.Equal(events, tt.want) {
					t.Errorf("events %v, want %v", events, tt.want)
				}
				events = nil
				if err := app.Run(RunWithArgs([]string{"protected"})); err != nil {
					t.Fatal(err)
				}
				if want := []string{"auth", "setup", "audit"}; !slices.Equal(events, want) {
					t.Errorf("protected command events %v, want %v", events, want)
				}
			}
		})
	}
}

func TestPreRunHooksSkipNonExecution(t *testing.T) {
	for _, args := range [][]string{
		{},
		{"--help"},
		{"-h"},
		{"--version"},
		{"help"},
		{"help", "cmd"},
		{"cmd", "--help"},
		{"help", "defaults"},
		{"defaults", "--help"},
		{"defaults", "list", "--help"},
		{"parent"},
		{"parent", "--help"},
		{"parent", "unknown"},
		{"cmd", "--unknown"},
		{"__complete", "cmd", ""},
		{"__completeNoDesc", "cmd", ""},
		{"__complete", "cmd", "--member", ""},
		{"completion", "bash"},
	} {
		t.Run(strings.Join(args, "/"), func(t *testing.T) {
			calls := 0
			app := newHookApplication(t, PreRunHook{Name: "auth", RunFunc: func(context.Context, *OutputWriter) error {
				calls++
				return errors.New("unexpected authentication")
			}})
			if err := app.AddCommand(&Command{
				Name: "cmd", Title: "Command", RunFunc: hookNoop,
				PreRunHooks: []PreRunHook{{Name: "local", RunFunc: func(context.Context, *OutputWriter) error {
					calls++
					return errors.New("unexpected command hook")
				}}},
				Args:  []Argument{{Name: "member", Choices: []string{"Alice"}}},
				Flags: &struct{ Member string }{},
			}, &Command{Name: "parent", Title: "Parent", PreRunHooks: []PreRunHook{{Name: "parent", RunFunc: func(context.Context, *OutputWriter) error {
				calls++
				return errors.New("unexpected parent hook")
			}}}, SubCommands: []*Command{{Name: "child", Title: "Child", RunFunc: hookNoop}}}); err != nil {
				t.Fatal(err)
			}
			err := app.Run(RunWithArgs(args))
			wantError := slices.Contains(args, "unknown") || slices.Contains(args, "--unknown")
			if (err != nil) != wantError {
				t.Errorf("error %v, want error: %v", err, wantError)
			}
			if calls != 0 {
				t.Errorf("hook called %d times without command execution", calls)
			}
		})
	}
}

func TestPreRunHookDefinitions(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	noop := PreRunHookFunc(func(context.Context, *OutputWriter) error { return nil })
	for _, tt := range []struct {
		name  string
		hooks []PreRunHook
		want  string
	}{
		{name: "empty name", hooks: []PreRunHook{{RunFunc: noop}}, want: "name cannot be empty"},
		{name: "blank name", hooks: []PreRunHook{{Name: " \t", RunFunc: noop}}, want: "name cannot be empty"},
		{name: "nil function", hooks: []PreRunHook{{Name: "auth"}}, want: "must have a RunFunc"},
		{name: "duplicate", hooks: []PreRunHook{{Name: "auth", RunFunc: noop}, {Name: "auth", RunFunc: noop}}, want: "duplicate pre-run hook name"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := NewApplication("hooktest", "Hook test", "v1.0.0", ApplicationWithPreRunHooks(tt.hooks...))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %v, want %q", err, tt.want)
			}
		})
	}
}

func TestPreRunHooksConfigurationFailure(t *testing.T) {
	calls := 0
	app := newHookApplication(t, PreRunHook{Name: "auth", RunFunc: func(context.Context, *OutputWriter) error {
		calls++
		return nil
	}})
	if err := app.AddCommand(&Command{Name: "cmd", Title: "Command", RunFunc: hookNoop}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("invalid: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(RunWithArgs([]string{"cmd", "--config", path})); err == nil {
		t.Fatal("expected configuration error")
	}
	if calls != 0 {
		t.Errorf("hook called %d times after configuration failure", calls)
	}
}

func TestSkippingPreRunHooksStillValidates(t *testing.T) {
	blocked := errors.New("flag validation failed")
	app := newHookApplication(t, PreRunHook{Name: "auth", RunFunc: func(context.Context, *OutputWriter) error {
		t.Error("skipped hook ran")
		return nil
	}})
	if err := app.AddCommand(&Command{
		Name: "public", Title: "Public", SkipPreRunHooks: []string{"auth"},
		ValidateFlagsFunc: func(context.Context) error { return blocked },
		RunFunc: func(context.Context, *Arguments, *OutputWriter) error {
			t.Error("command ran after validation failure")
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(RunWithArgs([]string{"public"})); err != blocked {
		t.Fatalf("expected original validation error, got %v", err)
	}
}

func TestPreRunHookOptionOrder(t *testing.T) {
	var events []string
	hook := func(name string) PreRunHook {
		return PreRunHook{Name: name, RunFunc: func(context.Context, *OutputWriter) error {
			events = append(events, name)
			return nil
		}}
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	app, _, err := NewApplication("hooktest", "Hook test", "v1.0.0", ApplicationWithWriter(io.Discard),
		ApplicationWithPreRunHooks(hook("auth")),
		ApplicationWithPreRunHooks(hook("setup"), hook("audit")))
	if err != nil {
		t.Fatal(err)
	}
	if err := app.AddCommand(&Command{Name: "cmd", Title: "Command", RunFunc: hookNoop}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(RunWithArgs([]string{"cmd"})); err != nil {
		t.Fatal(err)
	}
	if want := []string{"auth", "setup", "audit"}; !slices.Equal(events, want) {
		t.Errorf("events %v, want %v", events, want)
	}
}

func TestDefaultsCommandSkipPreRunHooks(t *testing.T) {
	for _, tt := range []struct {
		name        string
		commandName string
		opts        []ApplicationOptionFunc
		want        []string
	}{
		{
			name: "no opt-out", commandName: "defaults",
			want: []string{"auth", "setup", "audit"},
		},
		{
			name: "skip auth", commandName: "defaults",
			opts: []ApplicationOptionFunc{ApplicationWithDefaultsCommandSkipPreRunHooks("auth", "unknown")},
			want: []string{"setup", "audit"},
		},
		{
			name: "multiple options append", commandName: "defaults",
			opts: []ApplicationOptionFunc{
				ApplicationWithDefaultsCommandSkipPreRunHooks("auth"),
				ApplicationWithDefaultsCommandSkipPreRunHooks("setup"),
			},
			want: []string{"audit"},
		},
		{
			name: "renamed before skip", commandName: "preferences",
			opts: []ApplicationOptionFunc{
				ApplicationWithDefaultsCommandName("preferences"),
				ApplicationWithDefaultsCommandSkipPreRunHooks("auth"),
			},
			want: []string{"setup", "audit"},
		},
		{
			name: "renamed after skip", commandName: "preferences",
			opts: []ApplicationOptionFunc{
				ApplicationWithDefaultsCommandSkipPreRunHooks("auth"),
				ApplicationWithDefaultsCommandName("preferences"),
			},
			want: []string{"setup", "audit"},
		},
	} {
		for _, args := range [][]string{{"list"}, {"get", "team"}, {"set", "region", "eu"}, {"unset", "team"}} {
			t.Run(tt.name+"/"+args[0], func(t *testing.T) {
				t.Setenv("XDG_CONFIG_HOME", t.TempDir())
				var events []string
				var hooks []PreRunHook
				for _, name := range []string{"auth", "setup", "audit"} {
					hooks = append(hooks, PreRunHook{Name: name, RunFunc: func(context.Context, *OutputWriter) error {
						events = append(events, name)
						return nil
					}})
				}
				opts := append([]ApplicationOptionFunc{ApplicationWithWriter(io.Discard), ApplicationWithPreRunHooks(hooks...)}, tt.opts...)
				app, _, err := NewApplication("hooktest", "Hook test", "v1.0.0", opts...)
				if err != nil {
					t.Fatal(err)
				}
				config := filepath.Join(t.TempDir(), "config.yaml")
				if err := os.WriteFile(config, []byte("team: platform\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				invocation := append([]string{tt.commandName}, args...)
				invocation = append(invocation, "--config", config)
				if err := app.Run(RunWithArgs(invocation)); err != nil {
					t.Fatal(err)
				}
				if !slices.Equal(events, tt.want) {
					t.Errorf("events %v, want %v", events, tt.want)
				}

				// A matching leaf name outside the built-in subtree must not inherit skips.
				if err := app.AddCommand(&Command{Name: "custom", Title: "Custom", SubCommands: []*Command{
					{Name: args[0], Title: "Custom operation", RunFunc: hookNoop},
				}}); err != nil {
					t.Fatal(err)
				}
				events = nil
				if err := app.Run(RunWithArgs([]string{"custom", args[0]})); err != nil {
					t.Fatal(err)
				}
				if want := []string{"auth", "setup", "audit"}; !slices.Equal(events, want) {
					t.Errorf("custom command events %v, want %v", events, want)
				}
			})
		}
	}
}

func hookNoop(context.Context, *Arguments, *OutputWriter) error { return nil }
