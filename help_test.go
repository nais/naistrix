package naistrix_test

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nais/naistrix"
)

func TestHelpFormatting(t *testing.T) {
	for _, noColors := range []bool{false, true} {
		for _, args := range [][]string{
			{"--help"},
			{"help"},
			{"parent", "--help"},
			{"parent", "child", "--help"},
			{"help", "parent", "child"},
			{"child-alias", "--help"},
			{"defaults", "unset", "--help"},
		} {
			name := strings.Join(args, " ")
			if noColors {
				name += "/no-colors"
			}
			t.Run(name, func(t *testing.T) {
				var stdout bytes.Buffer
				app, _, err := naistrix.NewApplication("test", "An <info>application</info>.", "v0.0.0",
					naistrix.ApplicationWithWriter(&stdout),
					naistrix.ApplicationWithPreRunHooks(naistrix.PreRunHook{
						Name: "must-not-run",
						RunFunc: func(context.Context, *naistrix.OutputWriter) error {
							t.Error("help must not run hooks")
							return nil
						},
					}),
				)
				if err != nil {
					t.Fatal(err)
				}
				child := &naistrix.Command{
					Name:            "child",
					TopLevelAliases: []string{"child-alias"},
					Title:           "A <info>child</info> command.",
					Description:     "Use <info>--team <team></info>.\n\n<warn>Be careful.</warn> <error>Something failed.</error>\n\n<info>First line.\nSecond line.</info>\n\nKeep <env> visible.",
					Flags: &struct {
						Team string `usage:"Choose a <info>team</info>."`
					}{},
					RunFunc: completionMustNotRun(t),
				}
				if err := app.AddCommand(&naistrix.Command{
					Name: "parent", Title: "A <warn>parent</warn> command.",
					Description: "Parent <info>documentation</info>.", SubCommands: []*naistrix.Command{child},
				}); err != nil {
					t.Fatal(err)
				}
				invocation := append([]string{"--config", filepath.Join(t.TempDir(), "missing.yaml")}, args...)
				if noColors {
					invocation = append([]string{"--no-colors"}, invocation...)
				}
				if err := app.Run(naistrix.RunWithArgs(invocation)); err != nil {
					t.Fatal(err)
				}
				got := stdout.String()
				for _, unwanted := range []string{"<info>", "</info>", "<warn>", "</warn>", "<error>", "</error>"} {
					if strings.Contains(got, unwanted) {
						t.Errorf("help contains unrendered tag %q:\n%s", unwanted, got)
					}
				}
				if noColors && strings.Contains(got, "\x1b[") {
					t.Errorf("--no-colors help contains ANSI escape sequences: %q", got)
				}
				if args[0] == "parent" && len(args) > 2 || args[0] == "help" && len(args) > 1 {
					for _, want := range []string{"Use --team <team>.", "Be careful.", "Something failed.", "First line.\nSecond line.", "Keep <env> visible.", "Choose a team."} {
						if !strings.Contains(got, want) {
							t.Errorf("missing %q in help:\n%s", want, got)
						}
					}
				}
				// Help rendering must not mutate the source used by generated documentation.
				dir := t.TempDir()
				if err := app.GenerateDocs(naistrix.GenerateDocsWithTargetDir(dir)); err != nil {
					t.Fatal(err)
				}
				content := readFile(t, filepath.Join(dir, "test_parent_child.md"))
				if !strings.Contains(content, "Use `--team <team>`.") || !strings.Contains(content, "**Be careful.**") {
					t.Errorf("help rendering changed generated documentation:\n%s", content)
				}
			})
		}
	}
}
