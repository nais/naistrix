package naistrix

import (
	_ "embed"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/template"

	"github.com/spf13/pflag"
)

//go:embed templates/command.md.tmpl
var commandTemplate string

// generateDocsOptions holds the configuration options for generating documentation.
type generateDocsOptions struct {
	// targetDir is the directory where the generated documentation files will be saved, defaults to $CWD/docs.
	targetDir string
}

// GenerateDocsOptionFunc is a function that configures the documentation generation process.
type GenerateDocsOptionFunc func(*generateDocsOptions)

// GenerateDocsWithTargetDir can be used to specify a custom target dir to save the generated docs in.
func GenerateDocsWithTargetDir(targetDir string) GenerateDocsOptionFunc {
	return func(o *generateDocsOptions) {
		o.targetDir = targetDir
	}
}

// commandTemplateData holds all the data used to render the documentation for a single command.
type commandTemplateData struct {
	// Name is the command name.
	Name string

	// Aliases are local aliases for the command.
	Aliases []string

	// Description is a longer description of the command.
	Description string

	// Synopsis is the commands "useline".
	Synopsis string

	// Parent refers to the Name of the parent command, if any.
	Parent string

	// SubCommands is a list of subcommand names.
	SubCommands []string

	// LocalFlags are flags defined by the command.
	LocalFlags []commandTemplateDataFlag

	// InheritedFlags are flags defined by a parent command.
	InheritedFlags []commandTemplateDataFlag

	// Examples are examples on how to use the command.
	Examples []commandTemplateDataExample

	// TODO: deprecations, top level aliases, groups, more?
}

// commandTemplateDataFlag represents a command flag (option).
type commandTemplateDataFlag struct {
	// Name is the name of the flag, including the leading --.
	Name string

	// Short is the optional short name for the flag, including the leading -.
	Short string

	// Description is the description for the flag.
	Description string
}

// commandTemplateDataExample represents an example usage of a command.
type commandTemplateDataExample struct {
	// Description is the description of the example.
	Description string

	// Command is the example command.
	Command string
}

// GenerateDocs generates Markdown files for each command in the application.
//
// By default, the generated documentation will be placed in a "./docs" directory in the current directory. The target
// directory can be changed using the GenerateDocsWithTargetDir option.
func (a *Application) GenerateDocs(opts ...GenerateDocsOptionFunc) error {
	options := &generateDocsOptions{
		targetDir: "docs",
	}

	for _, o := range opts {
		o(options)
	}

	if err := os.MkdirAll(options.targetDir, 0o750); err != nil {
		return fmt.Errorf("failed to create target directory %q: %v", options.targetDir, err)
	}

	root, err := os.OpenRoot(options.targetDir)
	if err != nil {
		return fmt.Errorf("failed to open root directory %q: %v", options.targetDir, err)
	}
	defer func() {
		_ = root.Close()
	}()

	for _, cmd := range a.commands {
		if err := generateDocsForCommand(cmd, root); err != nil {
			return fmt.Errorf("failed to generate docs for command %q: %v", cmd.Name, err)
		}
	}

	return nil
}

// generateDocsForCommand generates docs for a command in a recursive manner.
func generateDocsForCommand(cmd *Command, root *os.Root) error {
	fn := filename(cmd)
	f, err := root.Create(fn)
	if err != nil {
		return fmt.Errorf("failed to create %q: %v", fn, err)
	}
	defer func() {
		_ = f.Close()
	}()

	var parent string
	if cmd.cobraCmd.HasParent() && cmd.cobraCmd.Parent() != cmd.cobraCmd.Root() {
		parent = cmd.cobraCmd.Parent().CommandPath()
	}

	if err := safeInitializeFlags(cmd); err != nil {
		return fmt.Errorf("failed to initialize flags: %w", err)
	}

	var synopsis string
	if len(cmd.SubCommands) > 0 {
		synopsis = cmd.cobraCmd.CommandPath() + " <command>"
	} else {
		synopsis = cmd.cobraCmd.UseLine()
	}

	data := commandTemplateData{
		Name:           cmd.cobraCmd.CommandPath(),
		Aliases:        commandTemplateAliases(cmd),
		Description:    cmd.Description,
		Synopsis:       synopsis,
		Parent:         parent,
		SubCommands:    commandTemplateSubCommands(cmd),
		Examples:       commandTemplateExamples(cmd),
		LocalFlags:     commandTemplateFlags(cmd.cobraCmd.LocalFlags()),
		InheritedFlags: commandTemplateFlags(cmd.cobraCmd.InheritedFlags()),
	}

	tmpl, err := template.New("command").Funcs(template.FuncMap{
		"linkify": func(s string) string {
			return strings.ReplaceAll(s, " ", "_")
		},
	}).Parse(commandTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	for _, s := range cmd.SubCommands {
		if err := generateDocsForCommand(s, root); err != nil {
			return fmt.Errorf("failed to generate docs for subcommand %q: %v", s.Name, err)
		}
	}

	return nil
}

// commandTemplateAliases generates a list of aliases for the given command.
func commandTemplateAliases(cmd *Command) []string {
	ret := make([]string, len(cmd.Aliases))
	for i, s := range cmd.Aliases {
		ret[i] = cmd.cobraCmd.CommandPath() + " " + s
	}
	slices.Sort(ret)
	return ret
}

// commandTemplateSubCommands generates a list of sub commands for the given command.
func commandTemplateSubCommands(cmd *Command) []string {
	ret := make([]string, len(cmd.SubCommands))
	for i, c := range cmd.SubCommands {
		ret[i] = c.cobraCmd.CommandPath()
	}
	slices.Sort(ret)
	return ret
}

// commandTemplateExamples generates a list of commandTemplateDataExample for the given command.
func commandTemplateExamples(cmd *Command) []commandTemplateDataExample {
	ret := make([]commandTemplateDataExample, len(cmd.Examples))
	for i, e := range cmd.Examples {
		ret[i] = commandTemplateDataExample{
			Description: e.Description,
			Command:     cmd.cobraCmd.CommandPath() + " " + e.Command,
		}
	}
	return ret
}

// commandTemplateFlags generates a list of commandTemplateDataFlag for the given flag set.
func commandTemplateFlags(flagSet *pflag.FlagSet) []commandTemplateDataFlag {
	ret := make([]commandTemplateDataFlag, 0)
	flagSet.VisitAll(func(f *pflag.Flag) {
		ret = append(ret, commandTemplateDataFlag{
			Name:        f.Name,
			Short:       f.Shorthand,
			Description: f.Usage,
		})
	})
	return ret
}

// filename generates a filename for the given command.
func filename(cmd *Command) string {
	return strings.ReplaceAll(cmd.cobraCmd.CommandPath(), " ", "_") + ".md"
}

// safeInitializeFlags triggers Cobra's flag merge for the command, which usually occurs runtime in Cobra. This might
// panic, so we recover and return an error instead.
func safeInitializeFlags(cmd *Command) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	_ = cmd.cobraCmd.LocalFlags()
	return nil
}
