package naistrix

import (
	"encoding/csv"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const redactedWizardValue = "<REDACTED>"

func (c *Command) wizardPreview(cmd *cobra.Command, args *Arguments, registry *commandRegistry) (string, bool, error) {
	tokens := strings.Split(cmd.CommandPath(), " ")
	var flags []string
	redacted := false
	var flagErr error
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		binding, ok := registry.flags[flag]
		if !ok || binding.ignored || flagErr != nil {
			return
		}
		values, err := wizardFlagValues(flag, binding.value)
		if err != nil {
			flagErr = err
			return
		}
		for _, value := range values {
			if strings.ContainsRune(value, 0) || !binding.sensitive && !canDisplayShellToken(value) {
				flagErr = fmt.Errorf("--%s contains a value that cannot be represented in a shell argument", flag.Name)
				return
			}
			if binding.sensitive {
				value = redactedWizardValue
				redacted = true
			}
			if flag.NoOptDefVal != "" {
				flags = append(flags, "--"+flag.Name+"="+value)
			} else {
				flags = append(flags, "--"+flag.Name, value)
			}
		}
	})
	if flagErr != nil {
		return "", false, flagErr
	}

	positional := args.All()
	separator := false
	for i, value := range positional {
		index := min(i, len(c.Args)-1)
		sensitive := index >= 0 && c.Args[index].Sensitive
		if strings.ContainsRune(value, 0) || !sensitive && !canDisplayShellToken(value) {
			return "", false, fmt.Errorf("an argument contains a value that cannot be represented in a shell argument")
		}
		if sensitive || strings.HasPrefix(value, "-") {
			separator = true
		}
		if sensitive {
			positional[i] = redactedWizardValue
			redacted = true
		}
	}
	if len(flags) == 0 && len(positional) == 0 {
		// With no explicit inputs, a copy pasted into a terminal would open the
		// wizard again. Use the built-in flag unless an application shadows it.
		const name = "non-interactive"
		if cmd.Flags().Lookup(name) != cmd.Root().PersistentFlags().Lookup(name) {
			return "", false, fmt.Errorf("--%s is shadowed and no input flags or arguments suppress the wizard", name)
		}
		flags = append(flags, "--"+name)
	}
	if separator {
		tokens = append(tokens, flags...)
		tokens = append(tokens, "--")
		tokens = append(tokens, positional...)
	} else {
		tokens = append(tokens, positional...)
		tokens = append(tokens, flags...)
	}
	for i, token := range tokens {
		if !canDisplayShellToken(token) {
			return "", false, fmt.Errorf("a command token cannot be displayed safely")
		}
		if i == 0 && shellCommandWordNeedsQuotes(token) {
			tokens[i] = singleQuoteShellToken(token)
		} else {
			tokens[i] = quoteShellToken(token)
		}
	}
	return strings.Join(tokens, " "), redacted, nil
}

func canDisplayShellToken(value string) bool {
	return utf8.ValidString(value) && strings.IndexFunc(value, func(r rune) bool {
		return unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' || unicode.Is(unicode.Bidi_Control, r)
	}) == -1
}

func shellCommandWordNeedsQuotes(value string) bool {
	if strings.ContainsRune(value, '=') {
		return true
	}
	switch value {
	case "if", "then", "else", "elif", "fi", "do", "done", "case", "esac", "while", "until", "for", "in", "function", "select", "time", "coproc", "repeat", "foreach", "end", "nocorrect":
		return true
	default:
		return false
	}
}

// wizardFlagValues reads the actual struct field, not the parser's cached value:
// wizard assignments to named string slices are not visible through pflag's copy.
func wizardFlagValues(flag *pflag.Flag, value reflect.Value) ([]string, error) {
	if value.Kind() == reflect.Slice {
		if flag.Value.Type() == "stringSlice" {
			// Viper reads the slice back through pflag's CSV representation.
			// A sole empty item is lost, and CSV normalizes embedded CRLFs.
			if value.Len() == 1 && value.Index(0).String() == "" {
				return nil, fmt.Errorf("--%s cannot represent a single empty value", flag.Name)
			}
			for i := range value.Len() {
				if strings.Contains(value.Index(i).String(), "\r\n") {
					return nil, fmt.Errorf("--%s cannot preserve embedded CRLF line endings", flag.Name)
				}
			}
		}
		if value.Len() == 0 {
			if flag.Value.Type() == "stringArray" {
				return nil, fmt.Errorf("--%s cannot represent an empty override", flag.Name)
			}
			return []string{""}, nil
		}
		values := make([]string, value.Len())
		for i := range value.Len() {
			item := value.Index(i).String()
			if flag.Value.Type() == "stringSlice" {
				// StringSlice parses each occurrence as CSV; quoting only at the
				// shell level would still split values containing commas.
				if item == "" {
					item = `""`
				} else {
					var csvValue strings.Builder
					writer := csv.NewWriter(&csvValue)
					if err := writer.Write([]string{item}); err != nil {
						return nil, fmt.Errorf("--%s cannot be serialized", flag.Name)
					}
					writer.Flush()
					item = strings.TrimSuffix(csvValue.String(), "\n")
				}
			}
			values[i] = item
		}
		return values, nil
	}
	var text string
	switch value.Kind() {
	case reflect.String:
		text = value.String()
	case reflect.Bool:
		text = strconv.FormatBool(value.Bool())
	case reflect.Int, reflect.Int64:
		if value.Type() == reflect.TypeFor[time.Duration]() {
			text = time.Duration(value.Int()).String()
		} else {
			text = strconv.FormatInt(value.Int(), 10)
		}
	case reflect.Uint:
		text = strconv.FormatUint(value.Uint(), 10)
	default:
		return nil, fmt.Errorf("--%s cannot be serialized", flag.Name)
	}
	return []string{text}, nil
}

// quoteShellToken targets POSIX shells and never allows shell expansion of data.
func quoteShellToken(value string) string {
	if value != "" && strings.IndexFunc(value, func(r rune) bool {
		return !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_@%+=:,./-", r)
	}) == -1 {
		return value
	}
	return singleQuoteShellToken(value)
}

func singleQuoteShellToken(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
