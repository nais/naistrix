package naistrix_test

import (
	"context"
	"strings"
	"testing"

	"github.com/nais/naistrix"
)

func TestApplicationValidation(t *testing.T) {
	defer func() {
		contains := "must have at least one command"
		if r := recover(); r == nil {
			t.Fatalf("expected panic for command with no name, but did not panic")
		} else if msg := r.(string); !strings.Contains(msg, contains) {
			t.Fatalf("expected panic message to contain %q, got: %q", contains, msg)
		}
	}()
	_, _ = (&naistrix.Application{}).Run(context.Background(), naistrix.Discard(), []string{})
}

func TestDuplicateCommandNamesAndAliases(t *testing.T) {
	t.Run("duplicate command names", func(t *testing.T) {
		app := &naistrix.Application{
			Name:  "test",
			Title: "Test Application",
			SubCommands: []*naistrix.Command{
				{
					Name:  "create",
					Title: "Create something",
				},
				{
					Name:  "create",
					Title: "Create something different",
				},
			},
		}

		defer func() {
			contains := "the application contains duplicate commands"
			if r := recover(); r == nil {
				t.Fatalf("expected panic")
			} else if msg := r.(string); !strings.Contains(msg, contains) {
				t.Fatalf("expected panic message to contain %q, got: %q", contains, msg)
			}
		}()
		_, _ = app.Run(context.Background(), naistrix.Discard(), []string{})
	})

	t.Run("duplicate alias", func(t *testing.T) {
		app := &naistrix.Application{
			Name:  "test",
			Title: "Test Application",
			SubCommands: []*naistrix.Command{
				{
					Name:    "create",
					Aliases: []string{"c"},
					Title:   "Create something",
				},
				{
					Name:    "count",
					Aliases: []string{"c"},
					Title:   "Count something",
				},
			},
		}

		defer func() {
			contains := "the application contains duplicate commands"
			if r := recover(); r == nil {
				t.Fatalf("expected panic")
			} else if msg := r.(string); !strings.Contains(msg, contains) {
				t.Fatalf("expected panic message to contain %q, got: %q", contains, msg)
			}
		}()
		_, _ = app.Run(context.Background(), naistrix.Discard(), []string{})
	})

	t.Run("duplicate name in sub-commands", func(t *testing.T) {
		app := &naistrix.Application{
			Name:  "test",
			Title: "Test Application",
			SubCommands: []*naistrix.Command{
				{
					Name:    "create",
					Aliases: []string{"c"},
					Title:   "Create something",
					SubCommands: []*naistrix.Command{
						{
							Name:    "car",
							Title:   "Create a car",
							Aliases: []string{"c"},
						},
						{
							Name:    "cat",
							Title:   "Create a cat",
							Aliases: []string{"c"},
						},
					},
				},
			},
		}

		defer func() {
			contains := `command "test create" contains duplicate commands`
			if r := recover(); r == nil {
				t.Fatalf("expected panic")
			} else if msg := r.(string); !strings.Contains(msg, contains) {
				t.Fatalf("expected panic message to contain %q, got: %q", contains, msg)
			}
		}()
		_, _ = app.Run(context.Background(), naistrix.Discard(), []string{})
	})
}
