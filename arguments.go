package naistrix

// Arguments represents the arguments sent to a command.
type Arguments struct {
	// args holds the command arguments provided by the user.
	args []*input
}

type input struct {
	name       string
	repeatable bool
	value      any
}

// newArguments creates a new Arguments instance based on the command definition and the arguments provided by the user.
func newArguments(commandArgs []Argument, userArgs []string) *Arguments {
	a := make([]*input, 0)

	for i, commandArg := range commandArgs {
		if i >= len(userArgs) {
			break
		}

		var v any
		if commandArg.Repeatable {
			v = userArgs[i:]
		} else {
			v = userArgs[i]
		}

		a = append(a, &input{
			name:       commandArg.Name,
			repeatable: commandArg.Repeatable,
			value:      v,
		})
	}

	return &Arguments{
		args: a,
	}
}

// Len returns the number of arguments.
func (a *Arguments) Len() int {
	return len(a.args)
}

// All returns the command arguments as a slice of strings.
func (a *Arguments) All() []string {
	ret := make([]string, 0)
	for _, arg := range a.args {
		if arg.repeatable {
			return append(ret, arg.value.([]string)...)
		} else {
			ret = append(ret, arg.value.(string))
		}
	}
	return ret
}

// Get retrieves a single argument by name. Using this for a repeatable argument or an argument that does not exist will
// cause a panic as a safeguard for the implementor.
func (a *Arguments) Get(name string) string {
	for _, arg := range a.args {
		if arg.name == name && !arg.repeatable {
			return arg.value.(string)
		}
	}
	panic(`"` + name + `" is not a valid argument`)
}

// GetRepeatable retrieves a single argument by name. Using this for a non-repeatable argument or an argument that does
// not exist will cause a panic as a safeguard for the implementor.
func (a *Arguments) GetRepeatable(name string) []string {
	for _, arg := range a.args {
		if arg.name == name && arg.repeatable {
			return arg.value.([]string)
		}
	}
	panic(`"` + name + `" is not a valid repeatable argument`)
}
