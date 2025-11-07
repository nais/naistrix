package naistrix

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	OutputVerbosityLevelNormal Count = iota
	OutputVerbosityLevelVerbose
	OutputVerbosityLevelDebug
	OutputVerbosityLevelTrace
)

// GlobalFlags defines flags that are global to all commands.
type GlobalFlags struct {
	// VerboseLevel indicates the verbosity level of the application.
	VerboseLevel Count `name:"verbose" short:"v" usage:"Set verbosity level. Use -v for verbose, -vv for debug, -vvv for trace."`

	// NoColors can be used to disable colored output.
	NoColors bool `name:"no-colors" usage:"Disable colors in the output."`

	// Config is the location of the configuration file.
	Config string `name:"config" usage:"Specify the location for the configuration file."`
}

// IsVerbose checks if the application is running in verbose mode (-v).
func (f GlobalFlags) IsVerbose() bool {
	return f.VerboseLevel > OutputVerbosityLevelNormal
}

// IsDebug checks if the application is running in debug mode (-vv).
func (f GlobalFlags) IsDebug() bool {
	return f.VerboseLevel > OutputVerbosityLevelVerbose
}

// IsTrace checks if the application is running in trace mode (-vvv or higher).
func (f GlobalFlags) IsTrace() bool {
	return f.VerboseLevel > OutputVerbosityLevelDebug
}

// Count is a type used for flags that when repeated increment a counter.
type Count int

// FlagAutoCompleter is an interface that can be implemented by flag values to provide auto-completion functionality.
type FlagAutoCompleter interface {
	AutoComplete(ctx context.Context, args *Arguments, toComplete string, flags any) (completions []string, activeHelp string)
}

// FileAutoCompleter is an interface that can be implemented by flag values to provide auto-completion functionality for
// a set of file extensions.
type FileAutoCompleter interface {
	FileExtensions() (extensions []string)
}

func setupFlag(name, short, usage string, value any, flags *pflag.FlagSet) error {
	if len(short) > 1 {
		return fmt.Errorf("short flag must be a single character")
	}

	if f := flags.Lookup(name); f != nil {
		return fmt.Errorf("duplicate flag name: %q", name)
	}

	switch ptr := value.(type) {
	case *string:
		if short == "" {
			flags.StringVar(ptr, name, *ptr, usage)
		} else {
			flags.StringVarP(ptr, name, short, *ptr, usage)
		}
	case *bool:
		if short == "" {
			flags.BoolVar(ptr, name, *ptr, usage)
		} else {
			flags.BoolVarP(ptr, name, short, *ptr, usage)
		}
	case *uint:
		if short == "" {
			flags.UintVar(ptr, name, *ptr, usage)
		} else {
			flags.UintVarP(ptr, name, short, *ptr, usage)
		}
	case *[]string:
		if short == "" {
			flags.StringSliceVar(ptr, name, *ptr, usage)
		} else {
			flags.StringSliceVarP(ptr, name, short, *ptr, usage)
		}
	case *int:
		if short == "" {
			flags.IntVar(ptr, name, *ptr, usage)
		} else {
			flags.IntVarP(ptr, name, short, *ptr, usage)
		}
	case *time.Duration:
		if short == "" {
			flags.DurationVar(ptr, name, *ptr, usage)
		} else {
			flags.DurationVarP(ptr, name, short, *ptr, usage)
		}
	case *Count:
		intPtr := (*int)(ptr)

		if short == "" {
			flags.CountVar(intPtr, name, usage)
		} else {
			flags.CountVarP(intPtr, name, short, usage)
		}
	default:
		return fmt.Errorf("unknown flag type: %T", value)
	}

	return nil
}

func setupFlags(cmd *cobra.Command, inputArgs []Argument, flags any, flagSet *pflag.FlagSet) error {
	if flags == nil {
		return nil
	}

	if err := validateFlags(flags); err != nil {
		return fmt.Errorf("invalid flags: %w", err)
	}

	re := regexp.MustCompile(`\|([^|]+)\|`)
	normalizeUsage := func(usage string) string {
		return re.ReplaceAllStringFunc(usage, func(s string) string {
			trimmed := strings.Trim(s, "|")
			return "`" + strings.ToUpper(trimmed) + "`"
		})
	}

	fields := reflect.TypeOf(flags).Elem()
	values := reflect.ValueOf(flags).Elem()
	for i := range fields.NumField() {
		field := fields.Field(i)
		value := values.Field(i)

		if !field.IsExported() || !value.CanAddr() {
			continue
		}

		if value.Kind() == reflect.Pointer && value.Elem().Kind() == reflect.Struct {
			continue
		}

		flagName, ok := field.Tag.Lookup("name")
		if !ok {
			flagName = strings.ToLower(field.Name)
		}

		flagUsage, ok := field.Tag.Lookup("usage")
		if !ok {
			flagUsage = field.Name
		}
		flagShort, _ := field.Tag.Lookup("short")

		actualValue := value.Addr().Interface()
		if err := setupFlag(flagName, flagShort, normalizeUsage(flagUsage), unwrap(actualValue), flagSet); err != nil {
			return fmt.Errorf("failed to setup flag %q: %w", flagName, err)
		}

		switch v := actualValue.(type) {
		case FlagAutoCompleter:
			_ = cmd.RegisterFlagCompletionFunc(
				flagName,
				func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
					completions, activeHelp := v.AutoComplete(cmd.Context(), newArguments(inputArgs, args), toComplete, flags)
					if activeHelp != "" {
						completions = cobra.AppendActiveHelp(completions, activeHelp)
					}
					return completions, cobra.ShellCompDirectiveNoFileComp
				},
			)
		case FileAutoCompleter:
			_ = cmd.RegisterFlagCompletionFunc(
				flagName,
				autocompleteFiles(v.FileExtensions()),
			)
		}
	}

	return nil
}

func unwrap(value any) any {
	v := reflect.ValueOf(value)
	switch v.Elem().Kind() {
	case reflect.String:
		var t *string
		return v.Convert(reflect.TypeOf(t)).Interface()
	default:
		return value
	}
}

// validateFlags is used to validate command flags.
func validateFlags(flags any) error {
	t := reflect.TypeOf(flags)

	if t.Kind() != reflect.Pointer {
		return fmt.Errorf("expected flags to be a pointer")
	}

	if t.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("expected flags to be a pointer to a struct")
	}

	return nil
}

// syncViperToFlags syncs values from Viper back to the flags struct.
// This ensures that values from config files and environment variables
// are reflected in the flags struct, not just CLI flag values.
func syncViperToFlags(flags any, viper interface {
	GetString(key string) string
	GetBool(key string) bool
	GetInt(key string) int
	GetUint(key string) uint
	GetDuration(key string) time.Duration
	GetStringSlice(key string) []string
	IsSet(key string) bool
},
) error {
	if flags == nil {
		return nil
	}

	fields := reflect.TypeOf(flags).Elem()
	values := reflect.ValueOf(flags).Elem()

	for i := range fields.NumField() {
		field := fields.Field(i)
		value := values.Field(i)

		if !field.IsExported() || !value.CanAddr() || !value.CanSet() {
			continue
		}

		// Handle embedded structs (like GlobalFlags)
		if value.Kind() == reflect.Pointer && value.Elem().Kind() == reflect.Struct {
			if err := syncViperToFlags(value.Interface(), viper); err != nil {
				return err
			}
			continue
		}

		flagName, ok := field.Tag.Lookup("name")
		if !ok {
			flagName = strings.ToLower(field.Name)
		}

		// Only update if Viper has a value for this key
		if !viper.IsSet(flagName) {
			continue
		}

		// Sync the value from Viper to the struct field
		switch value.Kind() {
		case reflect.String:
			value.SetString(viper.GetString(flagName))
		case reflect.Bool:
			value.SetBool(viper.GetBool(flagName))
		case reflect.Int, reflect.Int64:
			// Handle time.Duration
			if value.Type() == reflect.TypeOf(time.Duration(0)) {
				value.Set(reflect.ValueOf(viper.GetDuration(flagName)))
			} else {
				value.SetInt(int64(viper.GetInt(flagName)))
			}
		case reflect.Uint:
			value.SetUint(uint64(viper.GetUint(flagName)))
		case reflect.Slice:
			if value.Type().Elem().Kind() == reflect.String {
				value.Set(reflect.ValueOf(viper.GetStringSlice(flagName)))
			}
		}
	}

	return nil
}
