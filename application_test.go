package naistrix_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nais/naistrix"
)

var noop = func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error {
	return nil
}

func TestApplicationRunIsSingleUse(t *testing.T) {
	for _, mode := range []string{"success", "command error", "hook error", "configuration error", "flag validation error", "missing argument", "parse error", "unknown command", "help", "version", "completion"} {
		t.Run(mode, func(t *testing.T) {
			var output bytes.Buffer
			app, flags, err := naistrix.NewApplication("singleusetest", "Single-use tests", "v0.0.0", naistrix.ApplicationWithWriter(&output))
			if err != nil {
				t.Fatal(err)
			}
			flags.Config = filepath.Join(t.TempDir(), "config.yaml")
			failure := errors.New("test failure")
			runs := 0
			cmd := &naistrix.Command{
				Name: "cmd", Title: "Command",
				RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error {
					runs++
					if mode == "command error" {
						return failure
					}
					return nil
				},
			}
			args := []string{"cmd"}
			wantRuns, wantFailure := 0, false
			switch mode {
			case "success":
				wantRuns = 1
			case "command error":
				wantRuns, wantFailure = 1, true
			case "hook error":
				cmd.PreRunHooks = []naistrix.PreRunHook{{Name: "fail", RunFunc: func(context.Context, *naistrix.OutputWriter) error { return failure }}}
				wantFailure = true
			case "configuration error":
				if err := os.WriteFile(flags.Config, []byte("invalid: ["), 0o600); err != nil {
					t.Fatal(err)
				}
				wantFailure = true
			case "flag validation error":
				cmd.ValidateFlagsFunc = func(context.Context) error { return failure }
				wantFailure = true
			case "missing argument":
				cmd.Args = []naistrix.Argument{{Name: "value"}}
				cmd.DisableArgumentPrompts = true
				wantFailure = true
			case "parse error":
				args = append(args, "--unknown")
				wantFailure = true
			case "unknown command":
				args = []string{"unknown"}
				wantFailure = true
			case "help":
				args = append(args, "--help")
			case "version":
				args = []string{"--version"}
			case "completion":
				args = []string{"__complete", "cmd", ""}
			}
			if err := app.AddCommand(cmd); err != nil {
				t.Fatal(err)
			}
			err = app.Run(naistrix.RunWithArgs(args))
			if (err != nil) != wantFailure {
				t.Fatalf("unexpected first-run error: %v", err)
			}
			if (mode == "command error" || mode == "hook error" || mode == "flag validation error") && !errors.Is(err, failure) {
				t.Fatalf("first-run error was not preserved: %v", err)
			}
			firstOutput := output.String()
			firstCommand := strings.Join(app.ExecutedCommand(), " ")
			for range 2 {
				err := app.Run(naistrix.RunWithArgs([]string{"cmd", "--verbose"}))
				if !errors.Is(err, naistrix.ErrApplicationAlreadyRun) {
					t.Fatalf("expected ErrApplicationAlreadyRun, got %v", err)
				}
			}
			if runs != wantRuns {
				t.Errorf("RunFunc called %d times, want %d", runs, wantRuns)
			}
			if output.String() != firstOutput || strings.Join(app.ExecutedCommand(), " ") != firstCommand || flags.VerboseLevel != naistrix.OutputVerbosityLevelNormal {
				t.Error("a rejected run changed output, executed command, or flag values")
			}
		})
	}
}

func TestApplicationRunRejectsReentry(t *testing.T) {
	app, flags, err := naistrix.NewApplication("singleusetest", "Single-use tests", "v0.0.0", naistrix.ApplicationWithWriter(&bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}
	flags.Config = filepath.Join(t.TempDir(), "config.yaml")
	runs := 0
	if err := app.AddCommand(&naistrix.Command{
		Name: "cmd", Title: "Command",
		RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error {
			runs++
			if err := app.Run(naistrix.RunWithArgs([]string{"cmd"})); !errors.Is(err, naistrix.ErrApplicationAlreadyRun) {
				t.Errorf("expected reentry to be rejected, got %v", err)
			}
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(naistrix.RunWithArgs([]string{"cmd"})); err != nil {
		t.Fatal(err)
	}
	if runs != 1 {
		t.Errorf("RunFunc called %d times, want 1", runs)
	}
}

func TestApplicationRunAllowsDeprecatedReplacement(t *testing.T) {
	app, flags, err := naistrix.NewApplication("singleusetest", "Single-use tests", "v0.0.0", naistrix.ApplicationWithWriter(&bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}
	flags.Config = filepath.Join(t.TempDir(), "config.yaml")
	var executed []string
	if err := app.AddCommand(
		&naistrix.Command{
			Name: "old", Title: "Original command",
			RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error {
				executed = append(executed, "old")
				return &naistrix.DeprecatedCommandError{Replacement: []string{"replacement"}, ExecuteReplacement: true}
			},
		},
		&naistrix.Command{
			Name: "replacement", Title: "Replacement command",
			RunFunc: func(context.Context, *naistrix.Arguments, *naistrix.OutputWriter) error {
				executed = append(executed, "replacement")
				return nil
			},
		},
	); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(naistrix.RunWithArgs([]string{"old"})); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(executed, ","); got != "old,replacement" {
		t.Errorf("executed %q, want old,replacement", got)
	}
	if err := app.Run(naistrix.RunWithArgs([]string{"replacement"})); !errors.Is(err, naistrix.ErrApplicationAlreadyRun) {
		t.Errorf("expected subsequent run to be rejected, got %v", err)
	}
	if len(executed) != 2 {
		t.Error("a rejected run executed a command")
	}
}

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
		RunFunc: func(ctx context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
			userName := args.Get("user_name")
			out.Println("Hello, " + strings.ToUpper(userName) + "!")
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

		if contains := "version must not be empty"; !strings.Contains(err.Error(), contains) {
			t.Fatalf("expected error message to contain %q, got: %q", contains, err.Error())
		}
	})

	t.Run("top-level alias for top-level command", func(t *testing.T) {
		app, _, err := naistrix.NewApplication("app", "some app", "v0.0.0")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		err = app.AddCommand(&naistrix.Command{
			Name:            "cmd",
			Title:           "Some command",
			TopLevelAliases: []string{"alias"},
			RunFunc:         noop,
		})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if contains := "not allowed to have top-level aliases"; !strings.Contains(err.Error(), contains) {
			t.Fatalf("expected error message to contain %q, got: %q", contains, err.Error())
		}
	})

	t.Run("duplicate top-level aliases", func(t *testing.T) {
		app, _, err := naistrix.NewApplication("app", "some app", "v0.0.0")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		err = app.AddCommand(&naistrix.Command{
			Name:  "auth",
			Title: "Some auth command",
			SubCommands: []*naistrix.Command{{
				Name:            "login",
				Title:           "Login command",
				TopLevelAliases: []string{"l"},
				RunFunc:         noop,
			}, {
				Name:            "logout",
				Title:           "Logout command",
				TopLevelAliases: []string{"l"},
				RunFunc:         noop,
			}},
		})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if contains := `already registered by "login"`; !strings.Contains(err.Error(), contains) {
			t.Fatalf("expected error message to contain %q, got: %q", contains, err.Error())
		}
	})

	t.Run("duplicate top-level aliases and top-level command", func(t *testing.T) {
		app, _, err := naistrix.NewApplication("app", "some app", "v0.0.0")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		err = app.AddCommand(&naistrix.Command{
			Name:  "auth",
			Title: "Some auth command",
			SubCommands: []*naistrix.Command{{
				Name:            "login",
				Title:           "Login command",
				TopLevelAliases: []string{"login"},
				RunFunc:         noop,
			}},
		}, &naistrix.Command{
			Name:    "login",
			Title:   "Login command",
			RunFunc: noop,
		})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if contains := `duplicate commands and/or aliases: "login"`; !strings.Contains(err.Error(), contains) {
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
			RunFunc: noop,
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
					RunFunc: noop,
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
		app, _, err := naistrix.NewApplication(
			"app",
			"title",
			"v0.0.0",
			naistrix.ApplicationWithWriter(&bytes.Buffer{}),
		)
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
					RunFunc: noop,
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

	t.Run("duplicate command names in multiple calls to AddCommand", func(t *testing.T) {
		app, _, err := naistrix.NewApplication("test", "title", "v0.0.0")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		err = app.AddCommand(&naistrix.Command{
			Name:    "create",
			Title:   "Create something",
			RunFunc: noop,
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		err = app.AddCommand(&naistrix.Command{
			Name:    "create",
			Title:   "Create something different",
			RunFunc: noop,
		})
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
		RunFunc: func(ctx context.Context, _ *naistrix.Arguments, _ *naistrix.OutputWriter) error {
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
		RunFunc: noop,
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

func TestExecuteTopLevelAlias(t *testing.T) {
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

	type loginFlags struct {
		Username string
	}

	flags := &loginFlags{}

	err = app.AddCommand(&naistrix.Command{
		Name:  "auth",
		Title: "Auth command",
		SubCommands: []*naistrix.Command{
			{
				Name:            "login",
				Title:           "Login command",
				TopLevelAliases: []string{"login"},
				Flags:           flags,
				Args: []naistrix.Argument{
					{Name: "domain"},
				},
				RunFunc: func(_ context.Context, args *naistrix.Arguments, out *naistrix.OutputWriter) error {
					out.Println("Logged in: domain is", args.Get("domain"), "and username is", flags.Username)
					return nil
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if err := app.Run(naistrix.RunWithArgs([]string{"login", "example.com", "--username", "user"})); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	} else if expected := "Logged in: domain is example.com and username is user\n"; buf.String() != expected {
		t.Fatalf("expected version to be %q, got: %q", expected, buf.String())
	}
}
