package naistrix_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/nais/naistrix"
)

func TestStringArrayFlags(t *testing.T) {
	for _, tc := range []struct {
		name   string
		args   []string
		config string
		want   naistrix.StringArray
	}{
		{name: "defaults", want: naistrix.StringArray{"default,value"}},
		{
			name: "repeated flags preserve YAML and quotes",
			args: []string{"--set", "spec.env=[{name: LOG_LEVEL, value: debug}]", "--set", `spec.ingresses=["https://a.example.com", "https://b.example.com"]`},
			want: naistrix.StringArray{"spec.env=[{name: LOG_LEVEL, value: debug}]", `spec.ingresses=["https://a.example.com", "https://b.example.com"]`},
		},
		{name: "short flag", args: []string{"-s", "a,b"}, want: naistrix.StringArray{"a,b"}},
		{name: "empty value", args: []string{"--set="}, want: naistrix.StringArray{""}},
		{
			name:   "configuration",
			config: "set:\n  - 'a,b'\n  - ''\n",
			want:   naistrix.StringArray{"a,b", ""},
		},
		{
			name:   "CLI overrides configuration",
			config: "set:\n  - configured\n",
			args:   []string{"--set", "a,b"},
			want:   naistrix.StringArray{"a,b"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, _, err := naistrix.NewApplication("test", "Test application", "v0.0.0")
			if err != nil {
				t.Fatal(err)
			}
			flags := &struct {
				Set   naistrix.StringArray `short:"s"`
				Slice []string
			}{Set: naistrix.StringArray{"default,value"}}
			called := false
			err = app.AddCommand(&naistrix.Command{
				Name: "cmd", Title: "Test flags", Flags: flags,
				RunFunc: func(_ context.Context, _ *naistrix.Arguments, _ *naistrix.OutputWriter) error {
					called = true
					if !reflect.DeepEqual(flags.Set, tc.want) {
						t.Errorf("set = %#v, want %#v", flags.Set, tc.want)
					}
					if want := []string{"a", "b", "c"}; !reflect.DeepEqual(flags.Slice, want) {
						t.Errorf("slice = %#v, want %#v", flags.Slice, want)
					}
					return nil
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			configPath := filepath.Join(t.TempDir(), "config.yaml")
			if tc.config != "" {
				if err := os.WriteFile(configPath, []byte(tc.config), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			args := []string{"--config", configPath, "cmd", "--slice", "a,b", "--slice", "c"}
			if err := app.Run(naistrix.RunWithArgs(append(args, tc.args...))); err != nil {
				t.Fatal(err)
			}
			if !called {
				t.Fatal("command was not run")
			}
		})
	}
}

func TestStringArrayInheritedFlags(t *testing.T) {
	for _, scope := range []string{"global", "sticky"} {
		t.Run(scope, func(t *testing.T) {
			app, _, err := naistrix.NewApplication("test", "Test application", "v0.0.0")
			if err != nil {
				t.Fatal(err)
			}
			flags := &struct {
				Set naistrix.StringArray
			}{}
			called := false
			child := &naistrix.Command{
				Name: "child", Title: "Test inherited flags",
				RunFunc: func(_ context.Context, _ *naistrix.Arguments, _ *naistrix.OutputWriter) error {
					called = true
					want := naistrix.StringArray{"a,b", ""}
					if !reflect.DeepEqual(flags.Set, want) {
						t.Errorf("set = %#v, want %#v", flags.Set, want)
					}
					return nil
				},
			}
			parent := &naistrix.Command{Name: "parent", Title: "Parent", SubCommands: []*naistrix.Command{child}}
			if scope == "global" {
				if err := app.AddGlobalFlags(flags); err != nil {
					t.Fatal(err)
				}
			} else {
				parent.StickyFlags = flags
			}
			if err := app.AddCommand(parent); err != nil {
				t.Fatal(err)
			}
			args := []string{"--config", filepath.Join(t.TempDir(), "config.yaml"), "parent", "child", "--set", "a,b", "--set="}
			if err := app.Run(naistrix.RunWithArgs(args)); err != nil {
				t.Fatal(err)
			}
			if !called {
				t.Fatal("command was not run")
			}
		})
	}
}

func TestSetupFlag(t *testing.T) {
	t.Run("non-pointer", func(t *testing.T) {
		app, _, err := naistrix.NewApplication("test", "Test application", "v0.0.0")
		if err != nil {
			t.Fatalf("unexpected error when creating application: %v", err)
		}

		if err := app.AddGlobalFlags("foobar"); err == nil {
			t.Fatalf("expected error when adding invalid global flags type")
		} else if contains := "expected flags to be a pointer"; !strings.Contains(err.Error(), contains) {
			t.Fatalf("expected error message to contain %q, got: %q", contains, err.Error())
		}
	})

	t.Run("pointer to an invalid type", func(t *testing.T) {
		app, _, err := naistrix.NewApplication("test", "Test application", "v0.0.0")
		if err != nil {
			t.Fatalf("unexpected error when creating application: %v", err)
		}

		flags := "some string"
		if err := app.AddGlobalFlags(&flags); err == nil {
			t.Fatalf("expected error when adding invalid global flags type")
		} else if contains := "expected flags to be a pointer to a struct"; !strings.Contains(err.Error(), contains) {
			t.Fatalf("expected error message to contain %q, got: %q", contains, err.Error())
		}
	})

	t.Run("invalid short flag", func(t *testing.T) {
		app, _, err := naistrix.NewApplication("test", "Test application", "v0.0.0")
		if err != nil {
			t.Fatalf("unexpected error when creating application: %v", err)
		}

		flags := &struct {
			Quiet bool `short:"qu"`
		}{}

		if err := app.AddGlobalFlags(flags); err == nil {
			t.Fatalf("expected error when adding invalid global flags type")
		} else if contains := "short flag must be a single character"; !strings.Contains(err.Error(), contains) {
			t.Fatalf("expected error message to contain %q, got: %q", contains, err.Error())
		}
	})

	t.Run("unknown flag type", func(t *testing.T) {
		app, _, err := naistrix.NewApplication("test", "Test application", "v0.0.0")
		if err != nil {
			t.Fatalf("unexpected error when creating application: %v", err)
		}

		flags := &struct {
			Flag map[string]string
		}{}

		if err := app.AddGlobalFlags(flags); err == nil {
			t.Fatalf("expected error when adding invalid global flags type")
		} else if contains := "unknown flag type"; !strings.Contains(err.Error(), contains) {
			t.Fatalf("expected error message to contain %q, got: %q", contains, err.Error())
		}
	})

	t.Run("duplicate flags", func(t *testing.T) {
		app, _, err := naistrix.NewApplication("test", "Test application", "v0.0.0")
		if err != nil {
			t.Fatalf("unexpected error when creating application: %v", err)
		}

		flags := &struct {
			Verbose naistrix.Count `name:"verbose"`
		}{}

		if err := app.AddGlobalFlags(flags); err == nil {
			t.Fatalf("expected error when adding invalid global flags type")
		} else if contains := `duplicate flag name: "verbose"`; !strings.Contains(err.Error(), contains) {
			t.Fatalf("expected error message to contain %q, got: %q", contains, err.Error())
		}
	})
}
