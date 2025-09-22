package naistrix_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/nais/naistrix"
)

// Application with a single command that greets the user.
func ExampleApplication() {
	app, _, err := naistrix.NewApplication(
		"example",
		"example application",
		"v0.0.0",
	)
	if err != nil {
		panic(err)
	}

	err = app.AddCommand(&naistrix.Command{
		Name:  "greet",
		Title: "Greet the user",
		Args:  []naistrix.Argument{{Name: "user_name"}},
		RunFunc: func(ctx context.Context, out *naistrix.OutputWriter, args []string) error {
			out.Println("Hello, " + strings.ToUpper(args[0]) + "!")
			return nil
		},
	})
	if err != nil {
		panic(err)
	}

	_ = app.Run(naistrix.RunWithArgs([]string{"greet", "user"}))
	// Output: Hello, USER!
}

func TestApplicationValidation(t *testing.T) {
	t.Run("empty name", func(t *testing.T) {
		_, _, err := naistrix.NewApplication("", "", "v0.0.0")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if contains := "name must not be empty"; !strings.Contains(err.Error(), contains) {
			t.Fatalf("expected error message to contain %q, got: %q", contains, err.Error())
		}
	})

	t.Run("empty title", func(t *testing.T) {
		_, _, err := naistrix.NewApplication("example", "", "v0.0.0")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if contains := "title must not be empty"; !strings.Contains(err.Error(), contains) {
			t.Fatalf("expected error message to contain %q, got: %q", contains, err.Error())
		}
	})

	t.Run("name with spaces", func(t *testing.T) {
		_, _, err := naistrix.NewApplication("test app", "title", "v0.0.0")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if contains := "must not contain spaces"; !strings.Contains(err.Error(), contains) {
			t.Fatalf("expected error message to contain %q, got: %q", contains, err.Error())
		}
	})

	t.Run("empty version", func(t *testing.T) {
		_, _, err := naistrix.NewApplication("app", "title", "")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if contains := "must be a valid semantic version"; !strings.Contains(err.Error(), contains) {
			t.Fatalf("expected error message to contain %q, got: %q", contains, err.Error())
		}
	})

	t.Run("invalid version", func(t *testing.T) {
		_, _, err := naistrix.NewApplication("app", "title", "1.1.1")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if contains := "must be a valid semantic version"; !strings.Contains(err.Error(), contains) {
			t.Fatalf("expected error message to contain %q, got: %q", contains, err.Error())
		}
	})

	t.Run("no commands", func(t *testing.T) {
		app, _, err := naistrix.NewApplication("app", "title", "v0.0.0")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		err = app.Run()
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if contains := "must have at least one command"; !strings.Contains(err.Error(), contains) {
			t.Fatalf("expected error message to contain %q, got: %q", contains, err.Error())
		}
	})
}

func TestExecutedCommands(t *testing.T) {
	t.Run("single command", func(t *testing.T) {
		app, _, err := naistrix.NewApplication("app", "title", "v0.0.0")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		err = app.AddCommand(&naistrix.Command{
			Name:    "cmd",
			Title:   "Command",
			RunFunc: func(context.Context, *naistrix.OutputWriter, []string) error { return nil },
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if err := app.Run(naistrix.RunWithArgs([]string{"cmd"})); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		cmd := app.ExecutedCommand()
		if len(cmd) != 2 {
			t.Fatalf("expected 2 elements, got: %v", cmd)
		}

		if cmd[0] != "app" || cmd[1] != "cmd" {
			t.Fatalf("expected command to be [app cmd], got: %v", cmd)
		}
	})

	t.Run("nested command", func(t *testing.T) {
		app, _, err := naistrix.NewApplication("app", "title", "v0.0.0")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		err = app.AddCommand(&naistrix.Command{
			Name:  "cmd",
			Title: "Command",
			SubCommands: []*naistrix.Command{{
				Name:  "sub1",
				Title: "Sub Command 1",
				SubCommands: []*naistrix.Command{{
					Name:    "sub2",
					Title:   "Sub Command 2",
					RunFunc: func(context.Context, *naistrix.OutputWriter, []string) error { return nil },
				}},
			}},
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if err := app.Run(naistrix.RunWithArgs([]string{"cmd", "sub1", "sub2"})); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		cmd := app.ExecutedCommand()
		if len(cmd) != 4 {
			t.Fatalf("expected 4 elements, got: %v", cmd)
		}

		if cmd[0] != "app" || cmd[1] != "cmd" || cmd[2] != "sub1" || cmd[3] != "sub2" {
			t.Fatalf("expected command to be [app cmd sub1 sub2], got: %v", cmd)
		}
	})

	t.Run("invalid command", func(t *testing.T) {
		app, _, err := naistrix.NewApplication("app", "title", "v0.0.0")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		err = app.AddCommand(&naistrix.Command{
			Name:  "cmd",
			Title: "Command",
			SubCommands: []*naistrix.Command{{
				Name:  "sub1",
				Title: "Sub Command 1",
				SubCommands: []*naistrix.Command{{
					Name:    "sub2",
					Title:   "Sub Command 2",
					RunFunc: func(context.Context, *naistrix.OutputWriter, []string) error { return nil },
				}},
			}},
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if err := app.Run(naistrix.RunWithArgs([]string{"cmd", "sub1", "foo"})); err == nil {
			t.Fatalf("expected error")
		} else if contains := `unknown command "foo" for "app cmd sub1"`; !strings.Contains(err.Error(), contains) {
			t.Fatalf("expected error message to contain %q, got: %q", contains, err.Error())
		}

		if cmd := app.ExecutedCommand(); len(cmd) != 3 {
			t.Fatalf("expected 3 elements, got: %v", cmd)
		} else if cmd[0] != "app" || cmd[1] != "cmd" || cmd[2] != "sub1" {
			t.Fatalf("expected command to be [app cmd sub1], got: %v", cmd)
		}
	})
}

func TestDuplicateCommandNamesAndAliases(t *testing.T) {
	noop := func(context.Context, *naistrix.OutputWriter, []string) error { return nil }

	t.Run("duplicate command names", func(t *testing.T) {
		app, _, err := naistrix.NewApplication("test", "title", "v0.0.0")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		err = app.AddCommand(
			&naistrix.Command{
				Name:    "create",
				Title:   "Create something",
				RunFunc: noop,
			},
			&naistrix.Command{
				Name:    "create",
				Title:   "Create something different",
				RunFunc: noop,
			},
		)
		if err == nil {
			t.Fatalf("expected error, got nil")
		} else if contains := "the application contains duplicate commands"; !strings.Contains(err.Error(), contains) {
			t.Fatalf("expected error message to contain %q, got: %q", contains, err.Error())
		}
	})

	t.Run("duplicate alias", func(t *testing.T) {
		app, _, err := naistrix.NewApplication("test", "title", "v0.0.0")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		err = app.AddCommand(
			&naistrix.Command{
				Name:    "create",
				Aliases: []string{"c"},
				Title:   "Create something",
				RunFunc: noop,
			},
			&naistrix.Command{
				Name:    "count",
				Aliases: []string{"c"},
				Title:   "Count something",
				RunFunc: noop,
			},
		)
		if err == nil {
			t.Fatalf("expected error, got nil")
		} else if contains := "the application contains duplicate commands"; !strings.Contains(err.Error(), contains) {
			t.Fatalf("expected error message to contain %q, got: %q", contains, err.Error())
		}
	})

	t.Run("duplicate name in subcommands", func(t *testing.T) {
		app, _, err := naistrix.NewApplication("test", "title", "v0.0.0")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		err = app.AddCommand(&naistrix.Command{
			Name:    "create",
			Aliases: []string{"c"},
			Title:   "Create something",
			SubCommands: []*naistrix.Command{{
				Name:    "car",
				Title:   "Create a car",
				Aliases: []string{"c"},
				RunFunc: noop,
			}, {
				Name:    "cat",
				Title:   "Create a cat",
				Aliases: []string{"c"},
				RunFunc: noop,
			}},
		})
		if err == nil {
			t.Fatalf("expected error, got nil")
		} else if contains := `command "test create" contains duplicate commands`; !strings.Contains(err.Error(), contains) {
			t.Fatalf("expected error message to contain %q, got: %q", contains, err.Error())
		}
	})
}

type contextKeyType int

func TestRunWithContext(t *testing.T) {
	app, _, err := naistrix.NewApplication("app", "title", "v0.0.0")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	const contextKey contextKeyType = 0
	const contextValue = "value"

	err = app.AddCommand(&naistrix.Command{
		Name:  "cmd",
		Title: "Command",
		RunFunc: func(ctx context.Context, _ *naistrix.OutputWriter, _ []string) error {
			if actual := ctx.Value(contextKey); actual != contextValue {
				return fmt.Errorf("expected context value %q, got %q", contextValue, actual)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	ctx := context.WithValue(context.Background(), contextKey, contextValue)
	if err := app.Run(naistrix.RunWithContext(ctx), naistrix.RunWithArgs([]string{"cmd"})); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestApplicationVersion(t *testing.T) {
	buf := &bytes.Buffer{}
	app, _, err := naistrix.NewApplication(
		"app",
		"title",
		"v1.2.3",
		naistrix.ApplicationWithWriter(buf),
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	err = app.AddCommand(&naistrix.Command{
		Name:    "cmd",
		Title:   "Command",
		RunFunc: func(context.Context, *naistrix.OutputWriter, []string) error { return nil },
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if err := app.Run(naistrix.RunWithArgs([]string{"--version"})); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	} else if expected := "app version v1.2.3\n"; buf.String() != expected {
		t.Fatalf("expected version to be %q, got: %q", expected, buf.String())
	}
}
