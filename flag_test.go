package naistrix_test

import (
	"context"
	"strings"
	"testing"

	"github.com/nais/naistrix"
)

func TestSetupFlag(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid flags type", func(t *testing.T) {
		app := &naistrix.Application{
			Name:        "test",
			Title:       "Test application",
			StickyFlags: "foobar",
			SubCommands: []*naistrix.Command{
				{
					Name:  "cmd",
					Title: "Test command",
					RunFunc: func(context.Context, naistrix.Output, []string) error {
						return nil
					},
				},
			},
		}

		defer func() {
			contains := "expected flags to be a pointer to a struct"
			if r := recover(); r == nil {
				t.Fatalf("expected panic for invalid flags type")
			} else if msg := r.(string); !strings.Contains(msg, contains) {
				t.Fatalf("expected panic message to contain %q, got: %q", contains, msg)
			}
		}()
		_, _ = app.Run(ctx, naistrix.Discard(), []string{})
	})

	t.Run("non-pointer", func(t *testing.T) {
		app := &naistrix.Application{
			Name:        "test",
			Title:       "Test application",
			StickyFlags: struct{}{},
			SubCommands: []*naistrix.Command{
				{
					Name:  "cmd",
					Title: "Test command",
					RunFunc: func(context.Context, naistrix.Output, []string) error {
						return nil
					},
				},
			},
		}

		defer func() {
			contains := "expected flags to be a pointer to a struct"
			if r := recover(); r == nil {
				t.Fatalf("expected panic for invalid flags type")
			} else if msg := r.(string); !strings.Contains(msg, contains) {
				t.Fatalf("expected panic message to contain %q, got: %q", contains, msg)
			}
		}()
		_, _ = app.Run(ctx, naistrix.Discard(), []string{})
	})

	t.Run("invalid short flag", func(t *testing.T) {
		app := &naistrix.Application{
			Name:  "test",
			Title: "Test application",
			StickyFlags: &struct {
				Verbose naistrix.Count `short:"ver"`
			}{},
			SubCommands: []*naistrix.Command{
				{
					Name:  "cmd",
					Title: "Test command",
					RunFunc: func(context.Context, naistrix.Output, []string) error {
						return nil
					},
				},
			},
		}
		defer func() {
			contains := "short flag must be a single character"
			if r := recover(); r == nil {
				t.Fatalf("expected panic for invalid flags type")
			} else if msg := r.(string); !strings.Contains(msg, contains) {
				t.Fatalf("expected panic message to contain %q, got: %q", contains, msg)
			}
		}()
		_, _ = app.Run(ctx, naistrix.Discard(), []string{})
	})

	t.Run("unknown flag type", func(t *testing.T) {
		app := &naistrix.Application{
			Name:  "test",
			Title: "Test application",
			StickyFlags: &struct {
				Flag map[string]string
			}{},
			SubCommands: []*naistrix.Command{
				{
					Name:  "cmd",
					Title: "Test command",
					RunFunc: func(context.Context, naistrix.Output, []string) error {
						return nil
					},
				},
			},
		}
		defer func() {
			contains := "unknown flag type:"
			if r := recover(); r == nil {
				t.Fatalf("expected panic for invalid flags type")
			} else if msg := r.(string); !strings.Contains(msg, contains) {
				t.Fatalf("expected panic message to contain %q, got: %q", contains, msg)
			}
		}()
		_, _ = app.Run(ctx, naistrix.Discard(), []string{})
	})
}
