package naistrix

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"
)

func TestSensitiveArgumentPrompts(t *testing.T) {
	enableArgumentPrompts(t)
	for _, repeatable := range []bool{false, true} {
		t.Run(map[bool]string{false: "single", true: "repeatable"}[repeatable], func(t *testing.T) {
			app, _ := newWizardApplication(t)
			previousPlain, previousSecret := promptArgument, promptSecretArgument
			t.Cleanup(func() { promptArgument, promptSecretArgument = previousPlain, previousSecret })
			promptArgument = func(context.Context, string) (string, error) {
				t.Error("sensitive argument used an unmasked prompt")
				return "", io.EOF
			}
			answers := []string{"  secret  "}
			want := []string{"  secret  "}
			if repeatable {
				answers = append(answers, "second-secret", "")
				want = append(want, "second-secret")
			}
			promptSecretArgument = func(_ context.Context, prompt string) (string, error) {
				if len(answers) == 0 {
					t.Fatalf("unexpected prompt %q", prompt)
				}
				value := answers[0]
				answers = answers[1:]
				return value, nil
			}
			if err := app.AddCommand(&Command{
				Name: "create", Title: "Create", Args: []Argument{{Name: "secret", Sensitive: true, Repeatable: repeatable}},
				RunFunc: func(_ context.Context, args *Arguments, _ *OutputWriter) error {
					if !reflect.DeepEqual(args.All(), want) {
						t.Errorf("got %q, want %q", args.All(), want)
					}
					return nil
				},
			}); err != nil {
				t.Fatal(err)
			}
			if err := app.Run(RunWithArgs([]string{"create"})); err != nil {
				t.Fatal(err)
			}
			if len(answers) != 0 {
				t.Errorf("unused prompt answers: %q", answers)
			}
		})
	}
}

func TestSensitiveArgumentPromptError(t *testing.T) {
	enableArgumentPrompts(t)
	app, _ := newWizardApplication(t)
	previous := promptSecretArgument
	t.Cleanup(func() { promptSecretArgument = previous })
	promptSecretArgument = func(context.Context, string) (string, error) { return "", io.EOF }
	if err := app.AddCommand(&Command{
		Name: "create", Title: "Create", Args: []Argument{{Name: "secret", Sensitive: true}},
		RunFunc: func(context.Context, *Arguments, *OutputWriter) error {
			t.Error("RunFunc called after secret prompt error")
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(RunWithArgs([]string{"create"})); !errors.Is(err, io.EOF) {
		t.Errorf("expected prompt error, got %v", err)
	}
}
