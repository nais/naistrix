package naistrix

import (
	"reflect"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// commandRegistry shares command preparation and flag bindings between execution,
// wizard previews, and completion. Inherited callbacks use the actual target.
type commandRegistry struct {
	preparations map[*cobra.Command]completionPreparation
	commands     map[*cobra.Command]*Command
	flags        map[*pflag.Flag]flagBinding
	// wizardTarget records the decision once, after pre-run hooks, so all validators
	// and the execution callback agree on whether to defer validation.
	wizardTarget *cobra.Command
}

type completionPreparation struct {
	syncFlags     syncConfigAndFlagsFunc
	validateFlags ValidateFlagsFunc
}

type flagBinding struct {
	value     reflect.Value
	ignored   bool
	sensitive bool
}

func newCommandRegistry() *commandRegistry {
	return &commandRegistry{
		preparations: make(map[*cobra.Command]completionPreparation),
		commands:     make(map[*cobra.Command]*Command),
		flags:        make(map[*pflag.Flag]flagBinding),
	}
}
