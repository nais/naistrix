package naistrix_test

import (
	"context"
	"strings"
	"testing"

	"github.com/nais/naistrix"
)

// Application with a single command that greets the user.
func ExampleApplication() {
	app := &naistrix.Application{
		Name:  "example",
		Title: "Example Application",
		SubCommands: []*naistrix.Command{
			{
				Name:  "greet",
				Title: "Greet the user",
				Args: []naistrix.Argument{
					{Name: "user_name"},
				},
				RunFunc: func(ctx context.Context, out naistrix.Output, args []string) error {
					out.Println("Hello, " + strings.ToUpper(args[0]) + "!")
					return nil
				},
			},
		},
	}

	_ = app.Run(naistrix.RunWithArgs([]string{"greet", "user"}))
	// Output: Hello, USER!
}

func TestApplicationValidation(t *testing.T) {
	t.Run("no commands", func(t *testing.T) {
		defer func() {
			contains := "must have at least one command"
			if r := recover(); r == nil {
				t.Fatalf("expected panic for command with no name, but did not panic")
			} else if msg := r.(string); !strings.Contains(msg, contains) {
				t.Fatalf("expected panic message to contain %q, got: %q", contains, msg)
			}
		}()
		_ = (&naistrix.Application{Name: "app"}).Run(naistrix.RunWithOutput(naistrix.Discard()))
	})

	t.Run("empty name", func(t *testing.T) {
		defer func() {
			contains := "name must not be empty"
			if r := recover(); r == nil {
				t.Fatalf("expected panic for command with no name, but did not panic")
			} else if msg := r.(string); !strings.Contains(msg, contains) {
				t.Fatalf("expected panic message to contain %q, got: %q", contains, msg)
			}
		}()
		_ = (&naistrix.Application{}).Run(naistrix.RunWithOutput(naistrix.Discard()))
	})

	t.Run("name with spaces", func(t *testing.T) {
		defer func() {
			contains := "must not contain spaces"
			if r := recover(); r == nil {
				t.Fatalf("expected panic for command with no name, but did not panic")
			} else if msg := r.(string); !strings.Contains(msg, contains) {
				t.Fatalf("expected panic message to contain %q, got: %q", contains, msg)
			}
		}()
		_ = (&naistrix.Application{
			Name: "test app",
		}).Run(naistrix.RunWithOutput(naistrix.Discard()))
	})
}

func TestExecutedCommands(t *testing.T) {
	t.Run("single command", func(t *testing.T) {
		app := &naistrix.Application{
			Name: "app",
			SubCommands: []*naistrix.Command{
				{
					Name:    "cmd",
					Title:   "Command",
					RunFunc: func(context.Context, naistrix.Output, []string) error { return nil },
				},
			},
		}

		err := app.Run(naistrix.RunWithOutput(naistrix.Discard()), naistrix.RunWithArgs([]string{"cmd"}))
		if err != nil {
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
		app := &naistrix.Application{
			Name: "app",
			SubCommands: []*naistrix.Command{
				{
					Name:  "cmd",
					Title: "Command",
					SubCommands: []*naistrix.Command{
						{
							Name:  "sub1",
							Title: "Sub Command 1",
							SubCommands: []*naistrix.Command{
								{
									Name:    "sub2",
									Title:   "Sub Command 2",
									RunFunc: func(context.Context, naistrix.Output, []string) error { return nil },
								},
							},
						},
					},
				},
			},
		}

		err := app.Run(naistrix.RunWithOutput(naistrix.Discard()), naistrix.RunWithArgs([]string{"cmd", "sub1", "sub2"}))
		if err != nil {
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
		app := &naistrix.Application{
			Name: "app",
			SubCommands: []*naistrix.Command{
				{
					Name:  "cmd",
					Title: "Command",
					SubCommands: []*naistrix.Command{
						{
							Name:  "sub1",
							Title: "Sub Command 1",
							SubCommands: []*naistrix.Command{
								{
									Name:    "sub2",
									Title:   "Sub Command 2",
									RunFunc: func(context.Context, naistrix.Output, []string) error { return nil },
								},
							},
						},
					},
				},
			},
		}

		err := app.Run(naistrix.RunWithOutput(naistrix.Discard()), naistrix.RunWithArgs([]string{"cmd", "sub1", "foo"}))
		if err == nil {
			t.Fatalf("expected error")
		}

		cmd := app.ExecutedCommand()
		if len(cmd) != 3 {
			t.Fatalf("expected 3 elements, got: %v", cmd)
		}

		if cmd[0] != "app" || cmd[1] != "cmd" || cmd[2] != "sub1" {
			t.Fatalf("expected command to be [app cmd sub1], got: %v", cmd)
		}
	})
}

func TestDuplicateCommandNamesAndAliases(t *testing.T) {
	noop := func(context.Context, naistrix.Output, []string) error { return nil }

	t.Run("duplicate command names", func(t *testing.T) {
		app := &naistrix.Application{
			Name:  "test",
			Title: "Test Application",
			SubCommands: []*naistrix.Command{
				{
					Name:    "create",
					Title:   "Create something",
					RunFunc: noop,
				},
				{
					Name:    "create",
					Title:   "Create something different",
					RunFunc: noop,
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
		_ = app.Run(naistrix.RunWithOutput(naistrix.Discard()))
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
					RunFunc: noop,
				},
				{
					Name:    "count",
					Aliases: []string{"c"},
					Title:   "Count something",
					RunFunc: noop,
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
		_ = app.Run(naistrix.RunWithOutput(naistrix.Discard()))
	})

	t.Run("duplicate name in subcommands", func(t *testing.T) {
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
							RunFunc: noop,
						},
						{
							Name:    "cat",
							Title:   "Create a cat",
							Aliases: []string{"c"},
							RunFunc: noop,
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
		_ = app.Run(naistrix.RunWithOutput(naistrix.Discard()))
	})
}
