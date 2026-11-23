package naistrix

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nais/naistrix/wizardoption"
)

func TestQuoteShellToken(t *testing.T) {
	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("POSIX shell unavailable")
	}
	for _, value := range []string{
		"", "simple", "two words", "a'b", `a"b`, "$(printf injected)", "`printf injected`",
		"a; printf injected", "*?[abc]", "$HOME", "~", "line\nbreak", "秘密", "\\", "<REDACTED>",
	} {
		t.Run(value, func(t *testing.T) {
			output, err := exec.Command(shell, "-c", "printf '%s' "+quoteShellToken(value)).Output()
			if err != nil {
				t.Fatal(err)
			}
			if string(output) != value {
				t.Errorf("expected %q, got %q", value, output)
			}
		})
	}
}

func TestWizardPreviewRoundTrip(t *testing.T) {
	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("POSIX shell unavailable")
	}
	enableArgumentPrompts(t)
	type displayName string
	type resource string
	type resources []resource
	type createFlags struct {
		Display  displayName `name:"display-name"`
		Team     string
		Enabled  bool
		Replicas int
		Limit    uint
		Attempts Count
		Wait     time.Duration
		Labels   []string
		Resource resources
		Set      StringArray
	}
	initial := func() *createFlags {
		return &createFlags{
			Display: "before", Team: "before", Enabled: true, Replicas: 2, Limit: 9, Attempts: 1,
			Wait: time.Second, Labels: []string{"before"}, Resource: resources{"before"}, Set: StringArray{"before"},
		}
	}
	want := createFlags{
		Display: "Application's $(printf injected)", Team: "platform team", Enabled: false,
		Replicas: 0, Limit: 0, Attempts: 3, Wait: 1500 * time.Millisecond,
		Labels:   []string{"comma,inside", "line\nbreak", "", `quote"data`},
		Resource: resources{"postgres", "bucket"},
		Set:      StringArray{"A=x,y", "", `q="a,b"`, "line\r\nbreak"},
	}
	wantArgs := []string{"-my app", "value with 'quotes' and $(printf injected)", "", "line\nbreak"}
	definitions := []Argument{{Name: "name"}, {Name: "extra", Repeatable: true}}
	app, _ := newWizardApplication(t)
	flags := initial()
	var resolved *Arguments
	cmd := &Command{
		Name: "create", Title: "Create", Flags: flags, Args: definitions,
		WizardFunc: func(context.Context, *OutputWriter) ([]string, error) {
			*flags = want
			return wantArgs, nil
		},
		RunFunc: func(_ context.Context, args *Arguments, _ *OutputWriter) error { resolved = args; return nil },
	}
	if err := app.AddCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(RunWithArgs([]string{"create"})); err != nil {
		t.Fatal(err)
	}
	preview, redacted, err := cmd.wizardPreview(cmd.cobraCmd, resolved, app.registry)
	if err != nil || redacted {
		t.Fatalf("unexpected preview result: %q, %v, %v", preview, redacted, err)
	}
	if !strings.Contains(preview, " -- ") || !strings.Contains(preview, "--enabled=false") || !strings.Contains(preview, "--attempts=3") {
		t.Errorf("missing separator or explicit flag values: %s", preview)
	}
	output, err := exec.Command(shell, "-c", "printf '%s\\000' "+preview).Output()
	if err != nil {
		t.Fatal(err)
	}
	tokens := strings.Split(strings.TrimSuffix(string(output), "\x00"), "\x00")
	if tokens[0] != "wizardtest" {
		t.Fatalf("unexpected preview tokens: %q", tokens)
	}

	// Execute the preview against a fresh application with different input defaults.
	replayApp, _ := newWizardApplication(t)
	replayed := initial()
	if err := replayApp.AddCommand(&Command{
		Name: "create", Title: "Create", Flags: replayed, Args: definitions,
		WizardFunc: func(context.Context, *OutputWriter) ([]string, error) {
			t.Error("preview unexpectedly re-entered the wizard")
			return nil, nil
		},
		RunFunc: func(_ context.Context, args *Arguments, _ *OutputWriter) error {
			if !reflect.DeepEqual(args.All(), wantArgs) {
				t.Errorf("arguments changed: %q, want %q", args.All(), wantArgs)
			}
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := replayApp.Run(RunWithArgs(tokens[1:])); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*replayed, want) {
		t.Errorf("flags changed on replay:\n got: %#v\nwant: %#v\npreview: %s", *replayed, want, preview)
	}
}

func TestWizardPreviewRedaction(t *testing.T) {
	enableArgumentPrompts(t)
	app, output := newWizardApplication(t)
	flags := &struct {
		Token   string      `sensitive:"true"`
		Tokens  []string    `sensitive:"true"`
		Secrets StringArray `sensitive:"true"`
		Ignored string      `sensitive:"true" wizard:"ignore"`
	}{Token: "scalar-secret", Tokens: []string{"slice-secret-1", "slice-secret-2"}, Secrets: StringArray{"array-secret-1", "array-secret-2"}, Ignored: "ignored-secret"}
	wantArgs := []string{"argument-secret", "repeat-secret-1", "repeat-secret-2"}
	runs := 0
	cmd := &Command{
		Name: "create", Title: "Create", Flags: flags,
		Args:       []Argument{{Name: "secret", Sensitive: true}, {Name: "more", Repeatable: true, Sensitive: true}},
		WizardFunc: func(context.Context, *OutputWriter) ([]string, error) { return wantArgs, nil },
		RunFunc: func(_ context.Context, args *Arguments, _ *OutputWriter) error {
			runs++
			if !reflect.DeepEqual(args.All(), wantArgs) || flags.Token != "scalar-secret" {
				t.Error("redaction modified the actual inputs")
			}
			return nil
		},
	}
	if err := app.AddCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(RunWithArgs([]string{"create"})); err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"scalar-secret", "slice-secret-1", "slice-secret-2", "array-secret-1", "array-secret-2", "ignored-secret", "argument-secret", "repeat-secret-1", "repeat-secret-2", "--ignored"} {
		if strings.Contains(output.String(), secret) {
			t.Errorf("sensitive input leaked: %s", output)
		}
	}
	if count := strings.Count(output.String(), "'<REDACTED>'"); count != 8 {
		t.Errorf("expected 8 redacted values, got %d: %s", count, output)
	}
	if !strings.Contains(output.String(), "replace the redacted values") || !strings.Contains(output.String(), "--token '<REDACTED>'") || runs != 1 {
		t.Errorf("unexpected preview: %s; runs %d", output, runs)
	}
}

func TestWizardPreviewWithNoInputsUsesDiagnostics(t *testing.T) {
	enableArgumentPrompts(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var results, diagnostics bytes.Buffer
	app, _, err := NewApplication("wizardtest", "Wizard tests", "v0.0.0",
		ApplicationWithWriter(&results), ApplicationWithDiagnosticWriter(&diagnostics))
	if err != nil {
		t.Fatal(err)
	}
	if err := app.AddCommand(&Command{
		Name: "create", Title: "Create",
		WizardFunc: func(context.Context, *OutputWriter) ([]string, error) { return nil, nil },
		RunFunc: func(_ context.Context, _ *Arguments, out *OutputWriter) error {
			out.Println(`{"created":true}`)
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(RunWithArgs([]string{"create"})); err != nil {
		t.Fatal(err)
	}
	if got := results.String(); got != "{\"created\":true}\n" {
		t.Errorf("preview contaminated results: %q", got)
	}
	if !strings.Contains(diagnostics.String(), "wizardtest create --non-interactive\n") {
		t.Errorf("preview would re-enter the wizard on replay: %q", &diagnostics)
	}
}

func TestWizardPreviewQuotesShellCommandWord(t *testing.T) {
	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("POSIX shell unavailable")
	}
	enableArgumentPrompts(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	for _, name := range []string{"if", "x=y"} {
		t.Run(name, func(t *testing.T) {
			var diagnostics bytes.Buffer
			app, _, err := NewApplication(name, "Wizard tests", "v0.0.0", ApplicationWithDiagnosticWriter(&diagnostics))
			if err != nil {
				t.Fatal(err)
			}
			if err := app.AddCommand(&Command{
				Name: "create", Title: "Create",
				WizardFunc: func(context.Context, *OutputWriter) ([]string, error) { return nil, nil },
				RunFunc:    func(context.Context, *Arguments, *OutputWriter) error { return nil },
			}); err != nil {
				t.Fatal(err)
			}
			if err := app.Run(RunWithArgs([]string{"create"})); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(dir+"/"+name, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\"\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			_, preview, _ := strings.Cut(diagnostics.String(), "\n\n")
			cmd := exec.Command(shell, "-c", preview)
			cmd.Env = append(os.Environ(), "PATH="+dir)
			if output, err := cmd.CombinedOutput(); err != nil || string(output) != "create\n--non-interactive\n" {
				t.Fatalf("preview %q failed: %s (%v)", preview, output, err)
			}
		})
	}
}

func TestWizardPreviewUnavailableStillRuns(t *testing.T) {
	enableArgumentPrompts(t)
	for _, disabled := range []bool{false, true} {
		for _, reason := range []string{"empty array", "NUL flag", "NUL argument", "terminal escape"} {
			t.Run(reason+map[bool]string{false: "", true: " disabled"}[disabled], func(t *testing.T) {
				app, output := newWizardApplication(t)
				flags := &struct {
					Resources StringArray
					Token     string `sensitive:"true"`
				}{Resources: StringArray{"one"}, Token: "value"}
				args := []string{"app"}
				switch reason {
				case "empty array":
					flags.Resources = nil
				case "NUL flag":
					flags.Token = "secret\x00value"
				case "NUL argument":
					args[0] = "secret\x00value"
				case "terminal escape":
					args[0] = "\x1b[2J"
				}
				runs := 0
				cmd := &Command{
					Name: "create", Title: "Create", Flags: flags, Args: []Argument{{Name: "name"}},
					WizardFunc: func(context.Context, *OutputWriter) ([]string, error) { return args, nil },
					RunFunc:    func(context.Context, *Arguments, *OutputWriter) error { runs++; return nil },
				}
				if disabled {
					cmd.WizardOptions = []wizardoption.Option{wizardoption.WithoutCommandPreview()}
				}
				if err := app.AddCommand(cmd); err != nil {
					t.Fatal(err)
				}
				if err := app.Run(RunWithArgs([]string{"create"})); err != nil {
					t.Fatal(err)
				}
				if runs != 1 || strings.Contains(output.String(), "To run this non-interactively") || strings.Contains(output.String(), "secret") || strings.Contains(output.String(), "\x1b") {
					t.Errorf("runs %d, output %q", runs, output.String())
				}
				if strings.Contains(output.String(), "Command preview unavailable:") == disabled {
					t.Errorf("unexpected warning output: %q", output.String())
				}
			})
		}
	}
}

func TestWizardFlagTags(t *testing.T) {
	for _, flags := range []any{
		&struct {
			Team string `wizard:"invalid"`
		}{},
		&struct {
			Team string `wizard:""`
		}{},
		&struct {
			Token string `sensitive:"yes"`
		}{},
	} {
		app, _ := newWizardApplication(t)
		if err := app.AddCommand(&Command{
			Name: "create", Title: "Create", Flags: flags,
			RunFunc: func(context.Context, *Arguments, *OutputWriter) error { return nil },
		}); err == nil {
			t.Fatal("expected invalid flag tag to be rejected")
		}
	}
}

func TestWizardStringSlicePreviewLimitations(t *testing.T) {
	enableArgumentPrompts(t)
	for _, values := range [][]string{{""}, {"line\r\nbreak", "plain"}} {
		app, output := newWizardApplication(t)
		flags := &struct{ Labels []string }{}
		runs := 0
		if err := app.AddCommand(&Command{
			Name: "create", Title: "Create", Flags: flags,
			WizardFunc: func(context.Context, *OutputWriter) ([]string, error) {
				flags.Labels = values
				return nil, nil
			},
			RunFunc: func(context.Context, *Arguments, *OutputWriter) error { runs++; return nil },
		}); err != nil {
			t.Fatal(err)
		}
		if err := app.Run(RunWithArgs([]string{"create"})); err != nil {
			t.Fatal(err)
		}
		if runs != 1 || !strings.Contains(output.String(), "Command preview unavailable: --labels") {
			t.Errorf("runs %d, output %q", runs, output.String())
		}
	}
}

func TestWizardEnvironmentDefault(t *testing.T) {
	enableArgumentPrompts(t)
	app, output := newWizardApplication(t)
	t.Setenv("WIZARDTEST_TEAM", "environment-team")
	flags := &struct{ Team string }{}
	if err := app.AddCommand(&Command{
		Name: "create", Title: "Create", Flags: flags,
		WizardFunc: func(context.Context, *OutputWriter) ([]string, error) {
			if flags.Team != "environment-team" {
				t.Errorf("wizard saw team %q", flags.Team)
			}
			return nil, nil
		},
		RunFunc: func(context.Context, *Arguments, *OutputWriter) error { return nil },
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(RunWithArgs([]string{"create"})); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "--team environment-team") {
		t.Errorf("environment default missing from preview: %s", output)
	}
}

func TestWizardPreviewFlagShadowing(t *testing.T) {
	enableArgumentPrompts(t)
	app, output := newWizardApplication(t)
	local := &struct{ Team string }{Team: "local-team"}
	if err := app.AddCommand(&Command{
		Name: "parent", Title: "Parent",
		StickyFlags: &struct{ Team string }{Team: "parent-team"},
		SubCommands: []*Command{{
			Name: "create", Title: "Create", Flags: local,
			WizardFunc: func(context.Context, *OutputWriter) ([]string, error) {
				local.Team = "wizard-team"
				return nil, nil
			},
			RunFunc: func(context.Context, *Arguments, *OutputWriter) error { return nil },
		}},
	}); err != nil {
		t.Fatal(err)
	}
	// Global flags may be registered after command definitions, and shadowed
	// bindings must not override the actual execution target's field.
	if err := app.AddGlobalFlags(&struct{ Team string }{Team: "global-team"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(RunWithArgs([]string{"parent", "create"})); err != nil {
		t.Fatal(err)
	}
	if strings.Count(output.String(), "--team") != 1 || !strings.Contains(output.String(), "--team wizard-team") {
		t.Errorf("unexpected shadowed flag preview: %s", output)
	}
}

func TestSensitiveFlagSuppressesWizard(t *testing.T) {
	enableArgumentPrompts(t)
	app, output := newWizardApplication(t)
	flags := &struct {
		Token string `sensitive:"true"`
	}{}
	runs := 0
	if err := app.AddCommand(&Command{
		Name: "create", Title: "Create", Flags: flags,
		WizardFunc: func(context.Context, *OutputWriter) ([]string, error) {
			t.Error("explicit sensitive flag triggered wizard")
			return nil, nil
		},
		RunFunc: func(context.Context, *Arguments, *OutputWriter) error {
			runs++
			if flags.Token != "cli-secret" {
				t.Error("RunFunc did not receive the supplied secret")
			}
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(RunWithArgs([]string{"create", "--token", "cli-secret"})); err != nil {
		t.Fatal(err)
	}
	if runs != 1 || output.Len() != 0 {
		t.Errorf("runs %d, output %q", runs, output.String())
	}
}

func TestWizardPreviewEmptySliceOverride(t *testing.T) {
	enableArgumentPrompts(t)
	type flags struct {
		Labels []string
		Set    StringArray
	}
	initial := func() *flags { return &flags{Labels: []string{"configured"}, Set: StringArray{"configured"}} }
	app, _ := newWizardApplication(t)
	values := initial()
	var resolved *Arguments
	cmd := &Command{
		Name: "create", Title: "Create", Flags: values,
		WizardFunc: func(context.Context, *OutputWriter) ([]string, error) {
			values.Labels = nil
			values.Set = StringArray{""}
			return nil, nil
		},
		RunFunc: func(_ context.Context, args *Arguments, _ *OutputWriter) error { resolved = args; return nil },
	}
	if err := app.AddCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(RunWithArgs([]string{"create"})); err != nil {
		t.Fatal(err)
	}
	preview, _, err := cmd.wizardPreview(cmd.cobraCmd, resolved, app.registry)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preview, "--labels ''") || !strings.Contains(preview, "--set ''") {
		t.Fatalf("empty overrides missing from preview: %s", preview)
	}
	replay, _ := newWizardApplication(t)
	replayed := initial()
	if err := replay.AddCommand(&Command{
		Name: "create", Title: "Create", Flags: replayed,
		RunFunc: func(context.Context, *Arguments, *OutputWriter) error { return nil },
	}); err != nil {
		t.Fatal(err)
	}
	if err := replay.Run(RunWithArgs([]string{"create", "--labels", "", "--set", ""})); err != nil {
		t.Fatal(err)
	}
	if len(replayed.Labels) != 0 || !reflect.DeepEqual(replayed.Set, StringArray{""}) {
		t.Errorf("empty values changed: %#v", replayed)
	}
}
