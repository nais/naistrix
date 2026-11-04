package naistrix_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
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
		Examples: []naistrix.CommandExample{
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
		"title":                      "\n# myapp greet\n",
		"aliases heading":            "\n## Aliases\n",
		"options heading":            "\n## Options\n",
		"inherited options heading":  "\n## Inherited options\n",
		"examples heading":           "\n## Examples\n",
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

func TestGenerateDocs_FormattingTags(t *testing.T) {
	app := newTestApp(t)
	if err := app.AddGlobalFlags(&struct {
		Team string `usage:"Use <info>--team <team></info>. <warn>Choose carefully.</warn>"`
	}{}); err != nil {
		t.Fatal(err)
	}
	if err := app.AddCommand(&naistrix.Command{
		Name:  "greet",
		Title: "Greet someone",
		Description: strings.TrimSpace(heredoc.Doc(`
			Use <info>greet <name></info>.

			<warn>Be polite.</warn> <error>Do not shout.</error>

			Existing **Markdown** and [links](https://example.com) remain intact.

			A stray <info> tag and a mismatched <warn>pair</error> remain visible.
		`)),
		Flags: &struct {
			Loud bool `usage:"Print <info>LOUDLY</info>. <error>May disturb others.</error>"`
		}{},
		Examples: []naistrix.CommandExample{{
			Description: "Greet <info>Alice</info> <warn>politely</warn>, <error>not loudly</error>",
			Command:     "<name>",
		}},
		RunFunc: noop,
	}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := app.GenerateDocs(naistrix.GenerateDocsWithTargetDir(dir)); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, filepath.Join(dir, "myapp_greet.md"))
	for _, want := range []string{
		"Use `greet <name>`.\n\n**Be polite.** **Do not shout.**\n\nExisting **Markdown** and [links](https://example.com) remain intact.",
		": Print `LOUDLY`. **May disturb others.**",
		": Use `--team <team>`. **Choose carefully.**",
		"~~~bash\n# Greet Alice politely, not loudly\n$ myapp greet <name>\n",
		"A stray &lt;info&gt; tag and a mismatched &lt;warn&gt;pair&lt;/error&gt; remain visible.",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q in generated command docs:\n%s", want, content)
		}
	}
	for _, unwanted := range []string{"<info>", "</info>", "<warn>", "</warn>", "<error>", "</error>", "\x1b["} {
		if strings.Contains(content, unwanted) {
			t.Errorf("unexpected %q in generated command docs:\n%s", unwanted, content)
		}
	}
}

func TestGenerateDocs_DefaultsFormatting(t *testing.T) {
	for _, commandName := range []string{"defaults", "config"} {
		t.Run(commandName, func(t *testing.T) {
			app, _, err := naistrix.NewApplication("myapp", "Test defaults formatting", "v1.0.0",
				naistrix.ApplicationWithDefaultsCommandName(commandName),
			)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			if err := app.GenerateDocs(naistrix.GenerateDocsWithTargetDir(dir)); err != nil {
				t.Fatal(err)
			}
			for suffix, want := range map[string]string{
				"":          "Use `" + commandName + " describe <key>`",
				"_set":      "Set `<key>` to `<value>`",
				"_get":      "for each `<key>`",
				"_list":     "**Configured values reflect only the configuration file, not environment overrides or built-in flag defaults.**",
				"_describe": "configuration key `<key>`.",
				"_unset":    "Remove each `<key>`",
			} {
				content := readFile(t, filepath.Join(dir, "myapp_"+commandName+suffix+".md"))
				if !strings.Contains(content, want) {
					t.Errorf("missing %q in generated %s%s docs:\n%s", want, commandName, suffix, content)
				}
				for _, tag := range []string{"<info>", "</info>", "<warn>", "</warn>"} {
					if strings.Contains(content, tag) {
						t.Errorf("unconverted tag %q in generated %s%s docs:\n%s", tag, commandName, suffix, content)
					}
				}
			}
		})
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
	for _, heading := range []string{"\n# myapp auth\n", "\n## Available commands\n"} {
		if !strings.Contains(content, heading) {
			t.Errorf("expected heading %q, got\n%s", heading, content)
		}
	}
	if contains := "myapp auth <command>"; !strings.Contains(content, contains) {
		t.Errorf("expected file to contain %q, got\n%s", contains, content)
	}

	if contains := "has_children: true"; !strings.Contains(content, contains) {
		t.Errorf("expected file to contain %q, got\n%s", contains, content)
	}

	childContent := readFile(t, filepath.Join(dir, "myapp_auth_login.md"))
	if !strings.Contains(childContent, "\n# myapp auth login\n") {
		t.Errorf("expected child page to have a level-one heading, got\n%s", childContent)
	}
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

func TestGenerateDocs_ConfigurationKeys(t *testing.T) {
	for _, commandName := range []string{"defaults", "config"} {
		t.Run(commandName, func(t *testing.T) {
			app, flags, err := naistrix.NewApplication("myapp", "Test configuration docs", "v1.0.0",
				naistrix.ApplicationWithDefaultsCommandName(commandName),
				naistrix.ApplicationWithDefaultsDocumentation(
					naistrix.DefaultDefinition{Key: "zeta", Title: "<warn>Last</warn> setting with **Markdown**."},
					naistrix.DefaultDefinition{
						Key:   "ALPHA",
						Title: "First setting.",
						Description: strings.TrimSpace(heredoc.Doc(`
							Use <info>--alpha <value></info>.

							More detail with <error>emphasis</error>.

							Set it with <info>myapp defaults set alpha <value></info>. The <value> placeholder is required.
						`)),
					},
				),
			)
			if err != nil {
				t.Fatal(err)
			}
			if err := app.AddCommand(&naistrix.Command{
				Name:  "greet",
				Title: "Greet someone",
				Flags: &struct {
					First string `name:"alpha"`
					Zeta  string
				}{},
				RunFunc: noop,
			}); err != nil {
				t.Fatal(err)
			}
			flags.Config = filepath.Join(t.TempDir(), "config.yaml")
			// Docs must not read the user's config, even if it is malformed.
			contents := "private-local-key: [invalid yaml\n"
			if err := os.WriteFile(flags.Config, []byte(contents), 0o600); err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			if err := app.GenerateDocs(naistrix.GenerateDocsWithTargetDir(dir)); err != nil {
				t.Fatal(err)
			}
			listFile := "myapp_" + commandName + "_list.md"
			content := readFile(t, filepath.Join(dir, "configuration.md"))
			for _, want := range []string{
				"title: Configuration",
				"nav_order: 1.5",
				"\n# Configuration\n",
				"defaults for matching flags",
				"\n## Configuration keys\n",
				"Other keys may also be set.",
				"\n### alpha\n\nFirst setting.\n\nUse `--alpha <value>`.\n\nMore detail with **emphasis**.\n\nSet it with `myapp defaults set alpha <value>`. The &lt;value&gt; placeholder is required.",
				"\n### zeta\n\n**Last** setting with **Markdown**.",
			} {
				if !strings.Contains(content, want) {
					t.Errorf("expected %q in generated docs:\n%s", want, content)
				}
			}
			for _, unwanted := range []string{"<info>", "</info>", "<warn>", "</warn>", "<error>", "</error>", "\n#### ", "\n### `"} {
				if strings.Contains(content, unwanted) {
					t.Errorf("unexpected %q in generated configuration docs:\n%s", unwanted, content)
				}
			}
			if strings.Index(content, "\n### alpha\n") > strings.Index(content, "\n### zeta\n") {
				t.Errorf("configuration keys are not sorted:\n%s", content)
			}
			if strings.Contains(content, "\nparent:") {
				t.Errorf("configuration must be a top-level page:\n%s", content)
			}
			for _, subcommand := range []string{"set", "get", "list", "describe", "unset"} {
				link := "myapp_" + commandName + "_" + subcommand
				if !strings.Contains(content, "myapp "+commandName+" "+subcommand) || !strings.Contains(content, "]("+link+")") {
					t.Errorf("missing link to %s in configuration page:\n%s", subcommand, content)
				}
				if _, err := os.Stat(filepath.Join(dir, link+".md")); err != nil {
					t.Errorf("linked command page does not exist: %v", err)
				}
			}
			listContent := readFile(t, filepath.Join(dir, listFile))
			if strings.Contains(listContent, "[Configuration](configuration)") {
				t.Errorf("list page should not include a configuration backlink:\n%s", listContent)
			}
			files, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, file := range files {
				content := readFile(t, filepath.Join(dir, file.Name()))
				if strings.Contains(content, "private-local-key") || strings.Contains(content, "\x1b[") {
					t.Errorf("unexpected config data or terminal escape sequences in %s", file.Name())
				}
				if file.Name() != "configuration.md" && (strings.Contains(content, "## Configuration keys") || strings.Contains(content, "**Last**")) {
					t.Errorf("configuration keys should only appear on the configuration page, found in %s", file.Name())
				}
			}
			if got := readFile(t, flags.Config); got != contents {
				t.Errorf("doc generation modified configuration: %q", got)
			}
		})
	}
}

func TestGenerateDocs_NoConfigurationKeys(t *testing.T) {
	for _, commandName := range []string{"defaults", "config"} {
		t.Run(commandName, func(t *testing.T) {
			app, flags, err := naistrix.NewApplication("myapp", "Test configuration docs", "v1.0.0",
				naistrix.ApplicationWithDefaultsCommandName(commandName),
			)
			if err != nil {
				t.Fatal(err)
			}
			flags.Config = filepath.Join(t.TempDir(), "machine-specific", "private-config.yaml")
			dir := t.TempDir()
			if err := app.GenerateDocs(naistrix.GenerateDocsWithTargetDir(dir)); err != nil {
				t.Fatal(err)
			}
			content := readFile(t, filepath.Join(dir, "configuration.md"))
			for _, want := range []string{
				"title: Configuration",
				"nav_order: 1.5",
				"defaults for matching flags",
				"\n# Configuration\n",
				"\n## Configuration file location\n",
				"`$XDG_CONFIG_HOME/myapp/config.yaml`",
				"`~/.config/myapp/config.yaml`",
				"when `XDG_CONFIG_HOME` is not set",
				"`~/Library/Application Support/myapp/config.yaml`",
				`%AppData%\myapp\config.yaml`,
				"global `--config <path>` flag",
				"myapp " + commandName + " list --config /path/to/config.yaml",
				"Run `myapp " + commandName + " list` to see the actual configuration file path in use.",
				"\n## Managing configuration\n",
			} {
				if !strings.Contains(content, want) {
					t.Errorf("missing %q in configuration page:\n%s", want, content)
				}
			}
			if strings.Contains(content, flags.Config) || strings.Contains(content, "private-config.yaml") {
				t.Errorf("configuration page leaked a machine-specific path:\n%s", content)
			}
			if strings.Contains(content, "## Configuration keys") {
				t.Errorf("unexpected empty keys section:\n%s", content)
			}
			for _, subcommand := range []string{"set", "get", "list", "describe", "unset"} {
				link := "myapp_" + commandName + "_" + subcommand
				if !strings.Contains(content, "]("+link+")") {
					t.Errorf("missing link to %s in configuration page:\n%s", subcommand, content)
				}
				if _, err := os.Stat(filepath.Join(dir, link+".md")); err != nil {
					t.Errorf("linked command page does not exist: %v", err)
				}
			}
			content = readFile(t, filepath.Join(dir, "myapp_"+commandName+"_list.md"))
			if strings.Contains(content, "[Configuration](configuration)") {
				t.Errorf("list page should not include a configuration backlink:\n%s", content)
			}
		})
	}
}

func TestGenerateDocs_ValidatesDocumentedFlagsBeforeWriting(t *testing.T) {
	app, _, err := naistrix.NewApplication("myapp", "Test configuration docs", "v1.0.0",
		naistrix.ApplicationWithDefaultsDocumentation(naistrix.DefaultDefinition{Key: "team", Title: "Default team."}),
	)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "docs")
	err = app.GenerateDocs(naistrix.GenerateDocsWithTargetDir(dir))
	if want := `documented configuration key "team" does not match any registered flag`; err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("invalid documentation should not create the output directory: %v", err)
	}
	if err := app.AddGlobalFlags(&struct{ Team string }{}); err != nil {
		t.Fatal(err)
	}
	if err := app.GenerateDocs(naistrix.GenerateDocsWithTargetDir(dir)); err != nil {
		t.Fatalf("expected docs generation to succeed after registering the flag: %v", err)
	}
	// GenerateDocs must not consume the application invocation.
	if err := app.Run(naistrix.RunWithArgs([]string{"--help"})); err != nil {
		t.Fatalf("expected Run to remain available after doc generation: %v", err)
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
