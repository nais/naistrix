package naistrix

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestCommandPreRunHookOrderAndSkips(t *testing.T) {
	var events []string
	hook := func(name string) PreRunHook {
		return PreRunHook{Name: name, RunFunc: func(context.Context, *OutputWriter) error {
			events = append(events, name)
			return nil
		}}
	}
	app := newHookApplication(t, hook("global-1"), hook("global-2"))
	run := func(context.Context, *Arguments, *OutputWriter) error {
		events = append(events, "run")
		return nil
	}
	if err := app.AddCommand(&Command{
		Name: "parent", Title: "Parent", Aliases: []string{"p"},
		PreRunHooks:     []PreRunHook{hook("parent-1"), hook("parent-2")},
		SkipPreRunHooks: []string{"blocked"},
		SubCommands: []*Command{
			{Name: "child", Title: "Child", Aliases: []string{"c"}, TopLevelAliases: []string{"execute"}, PreRunHooks: []PreRunHook{hook("leaf")}, RunFunc: run},
			{Name: "public", Title: "Public", PreRunHooks: []PreRunHook{hook("leaf"), hook("public")}, SkipPreRunHooks: []string{"global-1", "parent-2", "leaf", "unknown"}, RunFunc: run},
			{Name: "blocked", Title: "Blocked", PreRunHooks: []PreRunHook{hook("blocked")}, RunFunc: run},
			{Name: "nested", Title: "Nested", PreRunHooks: []PreRunHook{hook("nested")}, SubCommands: []*Command{
				{Name: "child", Title: "Nested child", PreRunHooks: []PreRunHook{hook("leaf")}, RunFunc: run},
			}},
		},
	}, &Command{Name: "sibling", Title: "Sibling", PreRunHooks: []PreRunHook{hook("parent-1")}, RunFunc: run}); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		args []string
		want []string
	}{
		{args: []string{"parent", "child"}, want: []string{"global-1", "global-2", "parent-1", "parent-2", "leaf", "run"}},
		{args: []string{"p", "c"}, want: []string{"global-1", "global-2", "parent-1", "parent-2", "leaf", "run"}},
		{args: []string{"execute"}, want: []string{"global-1", "global-2", "parent-1", "parent-2", "leaf", "run"}},
		{args: []string{"parent", "public"}, want: []string{"global-2", "parent-1", "public", "run"}},
		{args: []string{"parent", "blocked"}, want: []string{"global-1", "global-2", "parent-1", "parent-2", "run"}},
		{args: []string{"parent", "nested", "child"}, want: []string{"global-1", "global-2", "parent-1", "parent-2", "nested", "leaf", "run"}},
		{args: []string{"sibling"}, want: []string{"global-1", "global-2", "parent-1", "run"}},
	} {
		t.Run(strings.Join(tt.args, "/"), func(t *testing.T) {
			// Use one application repeatedly to catch hooks or skips leaking between runs and branches.
			for range 2 {
				events = nil
				if err := app.Run(RunWithArgs(tt.args)); err != nil {
					t.Fatal(err)
				}
				if !slices.Equal(events, tt.want) {
					t.Errorf("events = %v, want %v", events, tt.want)
				}
			}
		})
	}
}

func TestCommandPreRunHooksBeforeInputHandling(t *testing.T) {
	for _, failAt := range []string{"", "parent", "leaf"} {
		t.Run("failure="+failAt, func(t *testing.T) {
			var events []string
			blocked := errors.New("setup failed")
			ctx := t.Context()
			app := newHookApplication(t)
			hook := func(name string) PreRunHook {
				return PreRunHook{Name: name, RunFunc: func(gotCtx context.Context, out *OutputWriter) error {
					events = append(events, name)
					if gotCtx != ctx || out != app.Output() {
						t.Error("hook did not receive execution context and application output")
					}
					if name == failAt {
						return blocked
					}
					return nil
				}}
			}
			if err := app.AddCommand(&Command{
				Name: "parent", Title: "Parent", PreRunHooks: []PreRunHook{hook("parent")},
				ValidateFlagsFunc: func(context.Context) error { events = append(events, "parent flags"); return nil },
				SubCommands: []*Command{{
					Name: "child", Title: "Child", PreRunHooks: []PreRunHook{hook("leaf"), hook("later")},
					ValidateFlagsFunc: func(context.Context) error { events = append(events, "leaf flags"); return nil },
					Args: []Argument{{Name: "member", ChoicesFunc: func(context.Context, *Arguments) ([]string, error) {
						events = append(events, "choices")
						return []string{"Alice"}, nil
					}}},
					ValidateArgsFunc: func(context.Context, *Arguments) error { events = append(events, "arguments"); return nil },
					RunFunc:          func(context.Context, *Arguments, *OutputWriter) error { events = append(events, "run"); return nil },
				}},
			}); err != nil {
				t.Fatal(err)
			}
			err := app.Run(RunWithContext(ctx), RunWithArgs([]string{"parent", "child", "Alice"}))
			want := []string{"parent", "leaf", "later", "parent flags", "leaf flags", "choices", "arguments", "run"}
			if failAt != "" {
				if err != blocked {
					t.Fatalf("error = %v, want original hook error", err)
				}
				want = []string{"parent"}
				if failAt == "leaf" {
					want = append(want, "leaf")
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(events, want) {
				t.Errorf("events = %v, want %v", events, want)
			}
		})
	}
}

func TestCommandPreRunHookDefinitions(t *testing.T) {
	noop := PreRunHook{Name: "auth", RunFunc: func(context.Context, *OutputWriter) error { return nil }}
	for _, tt := range []struct {
		name    string
		global  []PreRunHook
		command *Command
		want    string
	}{
		{name: "empty name", command: &Command{Name: "cmd", Title: "Command", RunFunc: hookNoop, PreRunHooks: []PreRunHook{{RunFunc: noop.RunFunc}}}, want: "name cannot be empty"},
		{name: "blank name", command: &Command{Name: "cmd", Title: "Command", RunFunc: hookNoop, PreRunHooks: []PreRunHook{{Name: " \t", RunFunc: noop.RunFunc}}}, want: "name cannot be empty"},
		{name: "nil function", command: &Command{Name: "cmd", Title: "Command", RunFunc: hookNoop, PreRunHooks: []PreRunHook{{Name: "auth"}}}, want: "must have a RunFunc"},
		{name: "local duplicate", command: &Command{Name: "cmd", Title: "Command", RunFunc: hookNoop, PreRunHooks: []PreRunHook{noop, noop}}, want: "duplicate pre-run hook name"},
		{name: "global duplicate", global: []PreRunHook{noop}, command: &Command{Name: "cmd", Title: "Command", RunFunc: hookNoop, PreRunHooks: []PreRunHook{noop}}, want: "duplicate pre-run hook name"},
		{name: "inherited duplicate", command: &Command{Name: "parent", Title: "Parent", PreRunHooks: []PreRunHook{noop}, SubCommands: []*Command{
			{Name: "child", Title: "Child", PreRunHooks: []PreRunHook{noop}, RunFunc: hookNoop},
		}}, want: `hooktest parent child`},
		{name: "skipped duplicate", global: []PreRunHook{noop}, command: &Command{Name: "cmd", Title: "Command", RunFunc: hookNoop, PreRunHooks: []PreRunHook{noop}, SkipPreRunHooks: []string{"auth"}}, want: "duplicate pre-run hook name"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := newHookApplication(t, tt.global...)
			registeredCommands, registeredDefinitions := len(app.commands), len(app.commandDefinitions)
			err := app.AddCommand(&Command{Name: "valid", Title: "Valid", RunFunc: hookNoop}, tt.command)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
			if len(app.commands) != registeredCommands || len(app.commandDefinitions) != registeredDefinitions {
				t.Error("invalid hooks partially registered the command batch")
			}
			if err := app.AddCommand(&Command{Name: "valid", Title: "Valid", RunFunc: hookNoop}); err != nil {
				t.Fatalf("failed registration retained command definitions: %v", err)
			}
		})
	}
}
