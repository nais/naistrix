package naistrix

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nais/naistrix/input"
	"github.com/nais/naistrix/wizardoption"
)

func newWizardApplication(t *testing.T) (*Application, *bytes.Buffer) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var output bytes.Buffer
	app, _, err := NewApplication("wizardtest", "Wizard tests", "v0.0.0", ApplicationWithWriter(io.Discard), ApplicationWithDiagnosticWriter(&output))
	if err != nil {
		t.Fatal(err)
	}
	return app, &output
}

func TestWizardTrigger(t *testing.T) {
	for _, tt := range []struct {
		name        string
		args        []string
		nonTerminal bool
		wizard      bool
		run         bool
		wantErr     bool
	}{
		{name: "bare command", args: []string{"parent", "create"}, wizard: true, run: true},
		{name: "argument", args: []string{"parent", "create", "app"}, run: true},
		{name: "local input flag", args: []string{"parent", "create", "--team", "other"}, wantErr: true},
		{name: "explicit false", args: []string{"parent", "create", "--enabled=false"}, wantErr: true},
		{name: "explicit zero", args: []string{"parent", "create", "--replicas=0"}, wantErr: true},
		{name: "local operational flag", args: []string{"parent", "create", "--trace"}, wizard: true, run: true},
		{name: "inherited input flag", args: []string{"parent", "create", "--tenant", "other"}, wantErr: true},
		{name: "inherited operational flag", args: []string{"parent", "create", "--quiet"}, wizard: true, run: true},
		{name: "global input flag", args: []string{"--region", "other", "parent", "create"}, wantErr: true},
		{name: "global operational flag", args: []string{"--operational", "parent", "create"}, wizard: true, run: true},
		{name: "built-in operational flags", args: []string{"parent", "create", "-v", "--no-colors"}, wizard: true, run: true},
		{name: "regular alias", args: []string{"parent", "c"}, wizard: true, run: true},
		{name: "top-level alias", args: []string{"new"}, wizard: true, run: true},
		{name: "help flag", args: []string{"parent", "create", "--help"}},
		{name: "help command", args: []string{"help", "parent", "create"}},
		{name: "completion", args: []string{"__complete", "parent", "create", ""}},
		{name: "parent help", args: []string{"parent"}},
		{name: "non-interactive", args: []string{"parent", "create"}, nonTerminal: true, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, output := newWizardApplication(t)
			previous := argumentsInteractive
			argumentsInteractive = func(context.Context) bool { return !tt.nonTerminal }
			t.Cleanup(func() { argumentsInteractive = previous })
			globals := &struct {
				Region      string
				Operational bool `wizard:"ignore"`
			}{Region: "default-region"}
			if err := app.AddGlobalFlags(globals); err != nil {
				t.Fatal(err)
			}
			flags := &struct {
				Team     string
				Enabled  bool
				Replicas int
				Trace    bool `wizard:"ignore"`
			}{Team: "default-team"}
			wizardCalls, runs := 0, 0
			leaf := &Command{
				Name: "create", Title: "Create", Aliases: []string{"c"}, TopLevelAliases: []string{"new"},
				Flags: flags, Args: []Argument{{Name: "name"}}, DisableArgumentPrompts: true,
				WizardFunc: func(context.Context, *OutputWriter) ([]string, error) {
					wizardCalls++
					return []string{"generated"}, nil
				},
				RunFunc: func(context.Context, *Arguments, *OutputWriter) error { runs++; return nil },
			}
			parent := &Command{
				Name: "parent", Title: "Parent", SubCommands: []*Command{leaf},
				StickyFlags: &struct {
					Tenant string
					Quiet  bool `wizard:"ignore"`
				}{Tenant: "default-tenant"},
			}
			if err := app.AddCommand(parent); err != nil {
				t.Fatal(err)
			}
			err := app.Run(RunWithArgs(tt.args))
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error: %v; output: %s", err, output)
			}
			if (wizardCalls == 1) != tt.wizard || wizardCalls > 1 {
				t.Errorf("wizard called %d times, expected triggered=%v", wizardCalls, tt.wizard)
			}
			if (runs == 1) != tt.run || runs > 1 {
				t.Errorf("RunFunc called %d times, expected executed=%v", runs, tt.run)
			}
			if strings.Contains(output.String(), "To run this non-interactively") != tt.wizard {
				t.Errorf("unexpected preview output: %s", output)
			}
			if tt.wizard && (strings.Contains(output.String(), "--trace") || strings.Contains(output.String(), "--quiet") || strings.Contains(output.String(), "--operational")) {
				t.Errorf("operational flags appeared in preview: %s", output)
			}
		})
	}
}

func TestWizardValidationOrderAndEffectiveDefaults(t *testing.T) {
	enableArgumentPrompts(t)
	app, output := newWizardApplication(t)
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("team: configured-team\nregion: configured-region\nreplicas: 3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	global := &struct{ Account string }{Account: "default-account"}
	if err := app.AddGlobalFlags(global); err != nil {
		t.Fatal(err)
	}
	inherited := &struct{ Region string }{}
	flags := &struct {
		Team     string
		Replicas int
	}{Replicas: 2}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var events []string
	leaf := &Command{
		Name: "create", Title: "Create", Flags: flags,
		WizardFunc: func(gotCtx context.Context, out *OutputWriter) ([]string, error) {
			events = append(events, "wizard")
			if gotCtx != ctx || out != app.Output() {
				t.Error("wizard did not receive execution context and output writer")
			}
			if flags.Team != "configured-team" || flags.Replicas != 3 || inherited.Region != "configured-region" || global.Account != "default-account" {
				t.Fatalf("wizard saw unsynchronized inputs: %+v, %+v, %+v", flags, inherited, global)
			}
			flags.Replicas = 4
			inherited.Region = "wizard-region"
			return []string{"MYAPP"}, nil
		},
		ValidateFlagsFunc: func(context.Context) error {
			events = append(events, "flags")
			if flags.Replicas != 4 || flags.Team != "configured-team" {
				t.Error("flag validation did not receive final values")
			}
			return nil
		},
		Args: []Argument{{
			Name: "name", ChoicesCaseInsensitive: true,
			ChoicesFunc: func(context.Context, *Arguments) ([]string, error) {
				events = append(events, "choices")
				if inherited.Region != "wizard-region" {
					t.Error("wizard value was overwritten before choices resolved")
				}
				return []string{"MyApp"}, nil
			},
		}},
		ValidateArgsFunc: func(_ context.Context, args *Arguments) error {
			events = append(events, "arguments")
			if args.Get("name") != "MyApp" {
				t.Error("arguments were not normalized")
			}
			return nil
		},
		RunFunc: func(_ context.Context, args *Arguments, _ *OutputWriter) error {
			events = append(events, "run")
			for _, token := range []string{"MyApp", "--team configured-team", "--region wizard-region", "--replicas 4", "--account default-account"} {
				if !strings.Contains(output.String(), token) {
					t.Errorf("preview missing %q before execution: %s", token, output)
				}
			}
			if strings.Contains(output.String(), "--config") || strings.Contains(output.String(), "--verbose") || strings.Contains(output.String(), "--no-colors") {
				t.Errorf("built-in operational flags appeared in preview: %s", output)
			}
			if args.Get("name") != "MyApp" || flags.Replicas != 4 || inherited.Region != "wizard-region" {
				t.Error("RunFunc received unexpected inputs")
			}
			return nil
		},
	}
	parent := &Command{
		Name: "parent", Title: "Parent", StickyFlags: inherited, SubCommands: []*Command{leaf},
		Args: []Argument{{Name: "name"}},
		ValidateFlagsFunc: func(context.Context) error {
			events = append(events, "parent flags")
			if inherited.Region != "wizard-region" {
				return errors.New("parent flag validator ran before wizard")
			}
			return nil
		},
		ValidateArgsFunc: func(context.Context, *Arguments) error {
			events = append(events, "parent arguments")
			return nil
		},
	}
	if err := app.AddCommand(parent); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(RunWithArgs([]string{"parent", "create", "--config", path}), RunWithContext(ctx)); err != nil {
		t.Fatal(err)
	}
	want := []string{"wizard", "parent flags", "flags", "parent arguments", "choices", "arguments", "run"}
	if !slices.Equal(events, want) {
		t.Errorf("events %v, want %v", events, want)
	}
	for _, name := range []string{"team", "replicas", "region", "account"} {
		if leaf.cobraCmd.Flags().Changed(name) {
			t.Errorf("wizard marked --%s as explicitly supplied", name)
		}
	}
}

func TestWizardRespectsNonInteractiveInvocation(t *testing.T) {
	for _, source := range []string{"flag", "environment", "context"} {
		t.Run(source, func(t *testing.T) {
			app, _ := newWizardApplication(t)
			terminalCtx := t.Context()
			ctx := terminalCtx
			args := []string{"create"}
			switch source {
			case "flag":
				args = append(args, "--non-interactive")
			case "environment":
				t.Setenv("WIZARDTEST_NON_INTERACTIVE", "true")
			case "context":
				ctx = input.WithNonInteractive(ctx)
			}
			previous := argumentsInteractive
			t.Cleanup(func() { argumentsInteractive = previous })
			// Simulate a terminal for the unmodified context only, ensuring the
			// wizard checks the invocation context rather than just the terminal.
			checks := 0
			argumentsInteractive = func(ctx context.Context) bool {
				checks++
				return ctx == terminalCtx || input.IsInteractive(ctx)
			}
			runs := 0
			if err := app.AddCommand(&Command{
				Name: "create", Title: "Create",
				WizardFunc: func(context.Context, *OutputWriter) ([]string, error) {
					t.Error("wizard ran when interaction was disabled")
					return nil, nil
				},
				RunFunc: func(context.Context, *Arguments, *OutputWriter) error { runs++; return nil },
			}); err != nil {
				t.Fatal(err)
			}
			if err := app.Run(RunWithArgs(args), RunWithContext(ctx)); err != nil {
				t.Fatal(err)
			}
			if runs != 1 || checks != 1 {
				t.Errorf("RunFunc called %d times, interaction checked %d times; want one of each", runs, checks)
			}
		})
	}
}

func TestWizardStopsOnErrors(t *testing.T) {
	enableArgumentPrompts(t)
	for _, failure := range []string{"callback", "missing arguments", "extra arguments", "undeclared arguments", "flags", "parent flags", "parent arguments", "choices", "arguments", "cancel before", "cancel after", "cancel during validation"} {
		t.Run(failure, func(t *testing.T) {
			app, output := newWizardApplication(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			failed := errors.New("test failure")
			if failure == "cancel before" {
				cancel()
			}
			calls := 0
			cmd := &Command{
				Name: "create", Title: "Create", Args: []Argument{{Name: "name"}},
				WizardFunc: func(context.Context, *OutputWriter) ([]string, error) {
					calls++
					switch failure {
					case "callback":
						return nil, failed
					case "missing arguments":
						return nil, nil
					case "extra arguments":
						return []string{"app", "extra"}, nil
					case "cancel after":
						cancel()
					}
					return []string{"app"}, nil
				},
				ValidateFlagsFunc: func(context.Context) error {
					if failure == "flags" {
						return failed
					}
					return nil
				},
				ValidateArgsFunc: func(context.Context, *Arguments) error {
					if failure == "cancel during validation" {
						cancel()
					}
					if failure == "arguments" {
						return failed
					}
					return nil
				},
				RunFunc: func(context.Context, *Arguments, *OutputWriter) error {
					t.Error("RunFunc called after an input error")
					return nil
				},
			}
			if failure == "choices" {
				cmd.Args[0].ChoicesFunc = func(context.Context, *Arguments) ([]string, error) { return nil, failed }
			}
			if failure == "undeclared arguments" {
				cmd.Args = nil
			}
			parent := &Command{
				Name: "parent", Title: "Parent", SubCommands: []*Command{cmd},
				ValidateFlagsFunc: func(context.Context) error {
					if failure == "parent flags" {
						return failed
					}
					return nil
				},
				ValidateArgsFunc: func(context.Context, *Arguments) error {
					if failure == "parent arguments" {
						return failed
					}
					return nil
				},
			}
			if err := app.AddCommand(parent); err != nil {
				t.Fatal(err)
			}
			previous := promptArgument
			promptArgument = func(context.Context, string) (string, error) {
				t.Error("automatic prompting ran after wizard")
				return "", failed
			}
			t.Cleanup(func() { promptArgument = previous })
			err := app.Run(RunWithArgs([]string{"parent", "create"}), RunWithContext(ctx))
			if err == nil {
				t.Fatal("expected an error")
			}
			if strings.HasPrefix(failure, "cancel") && !errors.Is(err, context.Canceled) {
				t.Errorf("expected cancellation, got %v", err)
			}
			if failure == "callback" && !errors.Is(err, failed) {
				t.Errorf("callback error was not preserved: %v", err)
			}
			if failure == "cancel before" && calls != 0 || failure != "cancel before" && calls != 1 {
				t.Errorf("unexpected wizard call count %d", calls)
			}
			if strings.Contains(output.String(), "To run this non-interactively") {
				t.Errorf("preview printed after failure: %s", output)
			}
		})
	}
}

func TestWizardWithoutCommandPreview(t *testing.T) {
	enableArgumentPrompts(t)
	app, output := newWizardApplication(t)
	runs := 0
	if err := app.AddCommand(&Command{
		Name: "create", Title: "Create",
		WizardOptions: []wizardoption.Option{wizardoption.WithoutCommandPreview()},
		WizardFunc:    func(context.Context, *OutputWriter) ([]string, error) { return nil, nil },
		RunFunc:       func(context.Context, *Arguments, *OutputWriter) error { runs++; return nil },
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(RunWithArgs([]string{"create"})); err != nil {
		t.Fatal(err)
	}
	if runs != 1 || output.Len() != 0 {
		t.Errorf("runs %d, output %q", runs, output.String())
	}
}

func TestWizardDefinitionValidation(t *testing.T) {
	noop := func(context.Context, *Arguments, *OutputWriter) error { return nil }
	wizard := func(context.Context, *OutputWriter) ([]string, error) { return nil, nil }
	for _, cmd := range []*Command{
		{Name: "missing-run", Title: "Missing run", WizardFunc: wizard},
		{Name: "parent", Title: "Parent", WizardFunc: wizard, SubCommands: []*Command{{Name: "child", Title: "Child", RunFunc: noop}}},
		{Name: "deprecated", Title: "Deprecated", WizardFunc: wizard, RunFunc: noop, Deprecated: &DeprecatedCommand{}},
		{Name: "options-without-wizard", Title: "Options without wizard", RunFunc: noop, WizardOptions: []wizardoption.Option{wizardoption.WithoutCommandPreview()}},
	} {
		t.Run(cmd.Name, func(t *testing.T) {
			app, _ := newWizardApplication(t)
			if err := app.AddCommand(cmd); err == nil {
				t.Fatal("expected command-definition error")
			}
		})
	}
}

func TestWizardPreRunHooks(t *testing.T) {
	enableArgumentPrompts(t)
	for _, invocation := range [][]string{{"parent", "create"}, {"parent", "c"}, {"new"}} {
		t.Run(strings.Join(invocation, " "), func(t *testing.T) {
			app, _ := newWizardApplication(t)
			flags := &struct{ Team string }{Team: "initial"}
			inherited := &struct{ Region string }{Region: "initial"}
			var events []string
			hook := func(name string) PreRunHook {
				return PreRunHook{Name: name, RunFunc: func(context.Context, *OutputWriter) error {
					events = append(events, name)
					if flags.Team != "configured" || inherited.Region != "configured" {
						t.Errorf("hook saw unsynchronized flags: %+v, %+v", flags, inherited)
					}
					return nil
				}}
			}
			ApplicationWithPreRunHooks(hook("application"), hook("skipped"))(app)
			t.Setenv("WIZARDTEST_TEAM", "configured")
			t.Setenv("WIZARDTEST_REGION", "configured")
			leaf := &Command{
				Name: "create", Title: "Create", Aliases: []string{"c"}, TopLevelAliases: []string{"new"},
				Flags: flags, Args: []Argument{{Name: "name"}},
				PreRunHooks: []PreRunHook{hook("leaf")},
				SkipPreRunHooksFunc: func() []string {
					events = append(events, "skips")
					if flags.Team != "configured" || inherited.Region != "configured" {
						t.Error("skip callback saw unsynchronized flags")
					}
					return []string{"skipped"}
				},
				WizardFunc: func(context.Context, *OutputWriter) ([]string, error) {
					events = append(events, "wizard")
					flags.Team, inherited.Region = "wizard", "wizard"
					return []string{"app"}, nil
				},
				ValidateFlagsFunc: func(context.Context) error {
					events = append(events, "leaf flags")
					if flags.Team != "wizard" || inherited.Region != "wizard" {
						t.Error("wizard values were overwritten")
					}
					return nil
				},
				RunFunc: func(context.Context, *Arguments, *OutputWriter) error {
					events = append(events, "run")
					return nil
				},
			}
			if err := app.AddCommand(&Command{
				Name: "parent", Title: "Parent", SubCommands: []*Command{leaf}, StickyFlags: inherited,
				PreRunHooks: []PreRunHook{hook("parent")},
				ValidateFlagsFunc: func(context.Context) error {
					events = append(events, "parent flags")
					if inherited.Region != "wizard" {
						t.Error("parent validator ran before wizard")
					}
					return nil
				},
			}); err != nil {
				t.Fatal(err)
			}
			if err := app.Run(RunWithArgs(invocation)); err != nil {
				t.Fatal(err)
			}
			want := []string{"skips", "application", "parent", "leaf", "wizard", "parent flags", "leaf flags", "run"}
			if !slices.Equal(events, want) {
				t.Errorf("events %v, want %v", events, want)
			}
		})
	}
}

func TestWizardPreRunHookError(t *testing.T) {
	enableArgumentPrompts(t)
	app, output := newWizardApplication(t)
	failure := errors.New("authentication failed")
	if err := app.AddCommand(&Command{
		Name: "create", Title: "Create",
		PreRunHooks: []PreRunHook{{Name: "auth", RunFunc: func(context.Context, *OutputWriter) error { return failure }}},
		WizardFunc: func(context.Context, *OutputWriter) ([]string, error) {
			t.Error("wizard ran after a hook error")
			return nil, nil
		},
		ValidateFlagsFunc: func(context.Context) error {
			t.Error("validation ran after a hook error")
			return nil
		},
		RunFunc: func(context.Context, *Arguments, *OutputWriter) error {
			t.Error("command ran after a hook error")
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(RunWithArgs([]string{"create"})); !errors.Is(err, failure) {
		t.Fatalf("expected original hook error, got %v", err)
	}
	if strings.Contains(output.String(), "To run this non-interactively") {
		t.Errorf("preview printed after a hook error: %s", output)
	}
}
