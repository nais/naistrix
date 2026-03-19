package naistrix_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nais/naistrix"
)

func TestGenerateDocs_FileContainsCorrectContent(t *testing.T) {
	type greetFlags struct {
		Loud bool `name:"loud" short:"l" usage:"Print loudly"`
	}

	app := newTestApp(t)
	if err := app.AddCommand(&naistrix.Command{
		Name:        "greet",
		Aliases:     []string{"hello", "hi"},
		Title:       "Greet someone",
		Description: "Prints a friendly greeting to the user.",
		Flags:       &greetFlags{},
		Args:        []naistrix.Argument{{Name: "name", Repeatable: true}},
		Examples: []naistrix.Example{
			{Description: "Greet Alice", Command: "Alice"},
		},
		RunFunc: noop,
	}); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	dir := filepath.Join(t.TempDir(), "docs")
	if err := app.GenerateDocs(naistrix.GenerateDocsWithTargetDir(dir)); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	content := readFile(t, filepath.Join(dir, "myapp_greet.md"))
	contains := map[string]string{
		"title":                      "## myapp greet",
		"description":                "Prints a friendly greeting to the user.",
		"synopsis":                   "myapp greet NAME [NAME...] [flags]",
		"hello alias":                "`myapp hello`",
		"hi alias":                   "`myapp hi`",
		"flag names":                 "`-l`, `--loud`",
		"flag description":           ": Print loudly",
		"inherited flag names":       "`-v`, `--verbose`",
		"inherited flag description": ": Set verbosity level",
		"example description":        "# Greet Alice",
		"example command":            "$ myapp greet Alice",
	}
	missing := false
	for k, v := range contains {
		if !strings.Contains(content, v) {
			missing = true
			t.Errorf("expected file to contain %q (%s)", v, k)
		}
	}
	if missing {
		t.Logf("Some errors occurred, the generated content:\n%s", content)
	}
}

func TestGenerateDocs_ParentCommand(t *testing.T) {
	app := newTestApp(t)
	if err := app.AddCommand(&naistrix.Command{
		Name:  "auth",
		Title: "Authentication commands",
		SubCommands: []*naistrix.Command{
			{Name: "login", Title: "Log in", RunFunc: noop},
			{Name: "logout", Title: "Log out", RunFunc: noop},
		},
	}); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	dir := t.TempDir()
	if err := app.GenerateDocs(naistrix.GenerateDocsWithTargetDir(dir)); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	for _, expected := range []string{"myapp_auth.md", "myapp_auth_login.md", "myapp_auth_logout.md"} {
		if _, err := os.Stat(filepath.Join(dir, expected)); os.IsNotExist(err) {
			t.Errorf("expected file %q to exist", expected)
		}
	}

	content := readFile(t, filepath.Join(dir, "myapp_auth.md"))
	if contains := "myapp auth <command>"; !strings.Contains(content, contains) {
		t.Errorf("expected file to contain %q, got\n%s", contains, content)
	}

	if contains := "has_children: true"; !strings.Contains(content, contains) {
		t.Errorf("expected file to contain %q, got\n%s", contains, content)
	}

	childContent := readFile(t, filepath.Join(dir, "myapp_auth_login.md"))
	if contains := "parent: myapp auth"; !strings.Contains(childContent, contains) {
		t.Errorf("expected file to contain %q, got\n%s", contains, childContent)
	}
}

func TestGenerateDocs_StrictModeOutputsMissingDescription(t *testing.T) {
	app := newTestApp(t)
	if err := app.AddCommand(&naistrix.Command{
		Name:    "cmd",
		Title:   "A command without description",
		RunFunc: noop,
	}); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	dir := t.TempDir()
	var buf bytes.Buffer
	if err := app.GenerateDocs(
		naistrix.GenerateDocsWithTargetDir(dir),
		naistrix.GenerateDocsWithStrictMode(),
		naistrix.GenerateDocsWithOutputWriter(&buf),
	); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if contains := `"myapp cmd" is missing a description`; !strings.Contains(buf.String(), contains) {
		t.Errorf("expected output to contain %q, got:\n%s", contains, buf.String())
	}
}

func newTestApp(t *testing.T) *naistrix.Application {
	t.Helper()
	app, _, err := naistrix.NewApplication("myapp", "My test application", "v1.0.0")
	if err != nil {
		t.Fatalf("failed to create application: %v", err)
	}
	return app
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %q: %v", path, err)
	}
	return string(data)
}
