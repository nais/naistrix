package naistrix_test

import (
	"context"
	"io"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/nais/naistrix"
	"github.com/nais/naistrix/input"
)

func TestCaseInsensitiveArgumentChoices(t *testing.T) {
	for _, tt := range []struct {
		name        string
		choices     []string
		insensitive bool
		repeatable  bool
		values      []string
		want        []string
		wantError   bool
	}{
		{name: "exact match by default", choices: []string{"foo", "bar"}, values: []string{"foo"}, want: []string{"foo"}},
		{name: "case sensitive by default", choices: []string{"foo", "bar"}, values: []string{"FOO"}, wantError: true},
		{name: "uppercase input", choices: []string{"foo", "bar"}, insensitive: true, values: []string{"FOO"}, want: []string{"foo"}},
		{name: "mixed case input", choices: []string{"foo", "bar"}, insensitive: true, values: []string{"bAr"}, want: []string{"bar"}},
		{name: "configured spelling", choices: []string{"Foo", "BAR"}, insensitive: true, values: []string{"foo"}, want: []string{"Foo"}},
		{name: "unicode folding", choices: []string{"S"}, insensitive: true, values: []string{"ſ"}, want: []string{"S"}},
		{name: "empty choice", choices: []string{""}, insensitive: true, values: []string{""}, want: []string{""}},
		{name: "no choices", insensitive: true, values: []string{"MiXeD"}, want: []string{"MiXeD"}},
		{name: "whitespace is not trimmed", choices: []string{"foo"}, insensitive: true, values: []string{" foo "}, wantError: true},
		{name: "configured whitespace is preserved", choices: []string{" Foo "}, insensitive: true, values: []string{" fOO "}, want: []string{" Foo "}},
		{name: "invalid choice", choices: []string{"foo", "bar"}, insensitive: true, values: []string{"BAZ"}, wantError: true},
		{name: "repeatable normalization", choices: []string{"Foo", "Bar"}, insensitive: true, repeatable: true, values: []string{"FOO", "bar", "fOo"}, want: []string{"Foo", "Bar", "Foo"}},
		{name: "repeatable no choices", insensitive: true, repeatable: true, values: []string{"FOO", "bar"}, want: []string{"FOO", "bar"}},
		{name: "repeatable remains case sensitive", choices: []string{"foo", "bar"}, repeatable: true, values: []string{"foo", "BAR"}, wantError: true},
		{name: "invalid repeatable choice", choices: []string{"foo", "bar"}, insensitive: true, repeatable: true, values: []string{"FOO", "BAZ"}, wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, _, err := naistrix.NewApplication("test", "Test", "v0.0.0", naistrix.ApplicationWithWriter(io.Discard))
			if err != nil {
				t.Fatal(err)
			}
			validated, called := false, false
			check := func(args *naistrix.Arguments) {
				t.Helper()
				if tt.wantError {
					t.Error("invalid choices reached custom validation or execution")
					return
				}
				if got := args.All(); !reflect.DeepEqual(got, tt.want) {
					t.Errorf("All() = %v, want %v", got, tt.want)
				}
				if tt.repeatable {
					if got := args.GetRepeatable("value"); !reflect.DeepEqual(got, tt.want) {
						t.Errorf("GetRepeatable() = %v, want %v", got, tt.want)
					}
				} else if got := args.Get("value"); got != tt.want[0] {
					t.Errorf("Get() = %q, want %q", got, tt.want[0])
				}
			}
			choices := slices.Clone(tt.choices)
			if err := app.AddCommand(&naistrix.Command{
				Name: "test", Title: "Test", DisableArgumentPrompts: true,
				Args: []naistrix.Argument{{Name: "value", Choices: choices, ChoicesCaseInsensitive: tt.insensitive, Repeatable: tt.repeatable}},
				ValidateFunc: func(_ context.Context, args *naistrix.Arguments) error {
					validated = true
					check(args)
					return nil
				},
				RunFunc: func(_ context.Context, args *naistrix.Arguments, _ *naistrix.OutputWriter) error {
					called = true
					check(args)
					return nil
				},
			}); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"test", "--config", filepath.Join(t.TempDir(), "config.yaml")}, tt.values...)
			original := slices.Clone(args)
			err = app.Run(naistrix.RunWithArgs(args))
			if tt.wantError {
				if err == nil || !strings.Contains(err.Error(), "invalid value") || validated || called {
					t.Fatalf("expected invalid choice before validation or execution, got %v, validated %v, called %v", err, validated, called)
				}
			} else if err != nil || !validated || !called {
				t.Fatalf("error %v, validated %v, called %v", err, validated, called)
			}
			if !slices.Equal(args, original) || !slices.Equal(choices, tt.choices) {
				t.Error("normalization changed caller-supplied arguments or configured choices")
			}
		})
	}
}

func TestAmbiguousArgumentChoices(t *testing.T) {
	for _, tt := range []struct {
		name        string
		choices     []string
		insensitive bool
		wantError   bool
	}{
		{name: "distinct casing allowed by default", choices: []string{"foo", "FOO"}},
		{name: "distinct insensitive choices", choices: []string{"foo", "BAR"}, insensitive: true},
		{name: "ambiguous casing", choices: []string{"foo", "FOO"}, insensitive: true, wantError: true},
		{name: "ambiguous unicode", choices: []string{"S", "ſ"}, insensitive: true, wantError: true},
		{name: "exact duplicate", choices: []string{"foo", "foo"}, insensitive: true, wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, _, err := naistrix.NewApplication("test", "Test", "v0.0.0")
			if err != nil {
				t.Fatal(err)
			}
			err = app.AddCommand(&naistrix.Command{
				Name: "test", Title: "Test",
				Args:    []naistrix.Argument{{Name: "value", Choices: tt.choices, ChoicesCaseInsensitive: tt.insensitive}},
				RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error { return nil },
			})
			if tt.wantError {
				if err == nil || !strings.Contains(err.Error(), "duplicate choices") {
					t.Fatalf("expected duplicate choices error, got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCaseInsensitiveChoicePrompt(t *testing.T) {
	choices := []string{"Upper", "Lower"}
	selections := 0
	restore := naistrix.SetChoicePromptForTest(func(_ string, got []string, _ ...input.SelectOptionFunc) (string, error) {
		selections++
		if !slices.Equal(got, choices) {
			t.Errorf("displayed choices %v, want %v", got, choices)
		}
		return got[0], nil
	})
	defer restore()
	app, _, err := naistrix.NewApplication("test", "Test", "v0.0.0", naistrix.ApplicationWithWriter(io.Discard))
	if err != nil {
		t.Fatal(err)
	}
	validated, called := false, false
	if err := app.AddCommand(&naistrix.Command{
		Name: "test", Title: "Test",
		Args: []naistrix.Argument{{Name: "value", Choices: choices, ChoicesCaseInsensitive: true}},
		ValidateFunc: func(_ context.Context, args *naistrix.Arguments) error {
			validated = true
			if got := args.Get("value"); got != "Upper" {
				t.Errorf("validated value %q, want Upper", got)
			}
			return nil
		},
		RunFunc: func(_ context.Context, args *naistrix.Arguments, _ *naistrix.OutputWriter) error {
			called = true
			if got := args.Get("value"); got != "Upper" {
				t.Errorf("executed value %q, want Upper", got)
			}
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(naistrix.RunWithArgs([]string{"test", "--config", filepath.Join(t.TempDir(), "config.yaml")})); err != nil {
		t.Fatal(err)
	}
	if selections != 1 || !validated || !called {
		t.Errorf("selections %d, validated %v, called %v", selections, validated, called)
	}
}
