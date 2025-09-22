package naistrix_test

import (
	"strings"
	"testing"

	"github.com/nais/naistrix"
)

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
