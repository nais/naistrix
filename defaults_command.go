package naistrix

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/nais/naistrix/internal/color"
	"github.com/spf13/viper"
)

// DefaultDefinition documents a configuration key. It neither assigns a default value nor restricts accepted values.
type DefaultDefinition struct {
	// Key must match the name of a registered flag. [Application.Run] validates this after registration. Keys are
	// case-insensitive and displayed in lowercase, matching configuration file keys.
	Key string

	// Title is a required short summary, shown by defaults list and shell completion. Whitespace is normalized to keep
	// it on one line. Aim for 80 visible characters or fewer; put explanations and examples in Description. This is
	// guidance, not a validated limit. Formatting tags are rendered by defaults list and stripped from shell completion.
	Title string

	// Description is optional detailed documentation, shown after Title by defaults describe and in generated docs.
	// It supports multiple paragraphs and formatting tags.
	Description string
}

func validateDefaultDefinitions(definitions []DefaultDefinition) error {
	seen := make(map[string]bool, len(definitions))
	for _, definition := range definitions {
		if definition.Key == "" || strings.ContainsFunc(definition.Key, unicode.IsSpace) {
			return fmt.Errorf("documented configuration key must not be empty or contain whitespace, got: %q", definition.Key)
		}
		key := strings.ToLower(definition.Key)
		if seen[key] {
			return fmt.Errorf("duplicate documented configuration key: %q", definition.Key)
		}
		seen[key] = true
		if strings.TrimSpace(definition.Title) == "" {
			return fmt.Errorf("documented configuration key %q is missing a title", definition.Key)
		}
	}
	return nil
}

// defaultsCommand creates the built-in defaults command for managing default flags for a user.
func defaultsCommand(commandName string, config *viper.Viper, definitions []DefaultDefinition) *Command {
	definitions = slices.Clone(definitions)
	for i := range definitions {
		definitions[i].Key = strings.ToLower(definitions[i].Key)
	}
	return &Command{
		Name:  commandName,
		Title: "Manage default flag values.",
		Description: heredoc.Docf(`
			The %[1]s command allows you to set, get, unset and list values stored in the configuration file.

			Configuration values act as defaults for various flags throughout the application.
			Use %[1]s list to see configured values and documented settings. Other keys may also be set.
			Use %[1]s describe <key> to read detailed documentation for a setting.
		`, commandName),
		SubCommands: []*Command{
			defaultsSet(config, definitions),
			defaultsGet(commandName, config),
			defaultsList(commandName, config, definitions),
			defaultsDescribe(config, definitions),
			defaultsUnset(config),
		},
	}
}

func defaultsSet(config *viper.Viper, definitions []DefaultDefinition) *Command {
	return &Command{
		Name: "set",
		Args: []Argument{
			{Name: "key"},
			{Name: "value"},
		},
		Title:       "Set a configuration value.",
		Description: "Set a configuration value in the configuration file. This value will be used as default for relevant flags throughout the application. Suggested keys include documented settings and existing configuration keys. Other keys may also be set.",
		AutoCompleteFunc: func(_ context.Context, args *Arguments, toComplete string) ([]string, string) {
			if args.Len() > 0 {
				return nil, ""
			}

			keys := make(map[string]string, len(definitions))
			for _, definition := range definitions {
				// Shell completion descriptions must be plain text on one line.
				description := strings.Join(strings.Fields(definition.Title), " ")
				keys[definition.Key] = color.StripTags(description)
			}
			settings, err := getSettingsFromConfigFile(config)
			if err != nil {
				return nil, err.Error()
			}
			for key := range settings {
				if _, ok := keys[key]; !ok {
					keys[key] = "Currently configured; no documentation available."
				}
			}
			var completions []string
			for _, key := range slices.Sorted(maps.Keys(keys)) {
				if strings.HasPrefix(key, toComplete) {
					completions = append(completions, key+"\t"+keys[key])
				}
			}

			return completions, "Choose a suggested key or enter another key"
		},
		RunFunc: func(_ context.Context, args *Arguments, out *OutputWriter) error {
			configFilePath := config.ConfigFileUsed()
			dir := filepath.Dir(configFilePath)

			if err := ensureDirectoryExists(dir); err != nil {
				return fmt.Errorf("unable to create directory %q for configuration file: %w", dir, err)
			}

			key := args.Get("key")
			value := args.Get("value")

			out.Printf("Set <info>%s</info> = <info>%s</info>\n", key, value)

			v := viper.New()
			v.SetConfigFile(configFilePath)
			if err := v.ReadInConfig(); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("unable to read configuration file %q: %w", configFilePath, err)
			}

			v.Set(key, value)
			if err := v.WriteConfig(); err != nil {
				return fmt.Errorf("unable to save configuration file: %w", err)
			}

			out.Println("Configuration file updated")
			return nil
		},
	}
}

func defaultsGet(defaultsCommandName string, config *viper.Viper) *Command {
	return &Command{
		Name:        "get",
		Title:       "Get one or more configuration values.",
		Description: "This command retrieves one or more configuration values from the configuration file.",
		Args: []Argument{{
			Name: "key", Repeatable: true, Prompt: "Select a configuration key", ChoicesFunc: configurationKeyChoices(config),
		}},
		RunFunc: func(_ context.Context, args *Arguments, out *OutputWriter) error {
			settings, err := getSettingsFromConfigFile(config)
			if err != nil {
				return fmt.Errorf("unable to read configuration file: %w", err)
			}

			for _, key := range args.GetRepeatable("key") {
				value, ok := settings[key]
				if !ok {
					out.Printf("No such configuration key: <info>%s</info>, create the value using <info>%s set %s <value></info>\n", key, defaultsCommandName, key)
					continue
				}

				out.Printf("<info>%s</info> = <info>%v</info>\n", key, value)

			}
			return nil
		},
	}
}

func defaultsList(defaultsCommandName string, config *viper.Viper, definitions []DefaultDefinition) *Command {
	return &Command{
		Name:        "list",
		Title:       "List configuration values.",
		Description: "List configured values and documented keys, including keys that have not been set. Configured values reflect only the configuration file, not environment overrides or built-in flag defaults. Other keys may also be set.",
		RunFunc: func(_ context.Context, _ *Arguments, out *OutputWriter) error {
			settings, err := getSettingsFromConfigFile(config)
			if err != nil {
				return fmt.Errorf("unable to read configuration file: %w", err)
			}

			if len(settings) == 0 && len(definitions) == 0 {
				out.Printf("The configuration file <info>%s</info> is empty, or it does not yet exist\n", config.ConfigFileUsed())
				out.Printf("Use the <info>%s set <key> <value></info> command to set configuration values\n", defaultsCommandName)
				return nil
			}

			descriptions := make(map[string]string, len(settings)+len(definitions))
			for key := range settings {
				descriptions[key] = "—"
			}
			for _, definition := range definitions {
				descriptions[definition.Key] = strings.Join(strings.Fields(definition.Title), " ")
			}
			headers := []string{"Key", "Configured value"}
			if len(definitions) > 0 {
				headers = append(headers, "Description")
			}
			values := [][]string{headers}
			for _, key := range slices.Sorted(maps.Keys(descriptions)) {
				value := "(not set)"
				if stored, ok := settings[key]; ok {
					value = fmt.Sprint(stored)
					if value == "" {
						value = `""`
					}
				}
				row := []string{key, value}
				if len(definitions) > 0 {
					row = append(row, descriptions[key])
				}
				values = append(values, row)
			}
			out.Printf("Configuration keys and values stored in <info>%s</info>:\n\n", config.ConfigFileUsed())
			if err := out.Table().Render(values); err != nil {
				return err
			}
			if len(definitions) > 0 {
				out.Println("\n(not set) means no value is stored in the configuration file. Other keys may also be set.")
				out.Printf("Use <info>%s describe <key></info> for detailed documentation.\n", defaultsCommandName)
			}
			out.Printf("\nUse the <info>%[1]s set <key> <value></info> command to update or create values, or the <info>%[1]s unset <key>[, <key>]</info> command to remove values\n", defaultsCommandName)
			return nil
		},
	}
}

func defaultsDescribe(config *viper.Viper, definitions []DefaultDefinition) *Command {
	return &Command{
		Name:        "describe",
		Title:       "Describe a configuration key.",
		Description: "Show the summary, detailed documentation, and configured value for a documented configuration key. Configured values reflect only the configuration file, not environment overrides or built-in flag defaults.",
		Args: []Argument{{
			Name: "key", Prompt: "Select a documented configuration key", ChoicesCaseInsensitive: true,
			ChoicesFunc: func(context.Context, *Arguments) ([]string, error) {
				keys := make([]string, len(definitions))
				for i, definition := range definitions {
					keys[i] = definition.Key
				}
				slices.Sort(keys)
				return keys, nil
			},
		}},
		AutoCompleteFunc: func(_ context.Context, args *Arguments, toComplete string) ([]string, string) {
			if args.Len() > 0 {
				return nil, ""
			}
			var completions []string
			for _, definition := range definitions {
				if strings.HasPrefix(definition.Key, strings.ToLower(toComplete)) {
					title := color.StripTags(strings.Join(strings.Fields(definition.Title), " "))
					completions = append(completions, definition.Key+"\t"+title)
				}
			}
			slices.Sort(completions)
			return completions, "Choose a documented configuration key"
		},
		RunFunc: func(_ context.Context, args *Arguments, out *OutputWriter) error {
			settings, err := getSettingsFromConfigFile(config)
			if err != nil {
				return fmt.Errorf("unable to read configuration file: %w", err)
			}
			key := args.Get("key")
			for _, definition := range definitions {
				if definition.Key != key {
					continue
				}
				value := "(not set)"
				if stored, ok := settings[key]; ok {
					value = fmt.Sprint(stored)
					if value == "" {
						value = `""`
					}
				}
				out.Printf("Configuration key: <info>%s</info>\n", key)
				out.Printf("Configured value: <info>%s</info>\n\n", value)
				out.Println(strings.Join(strings.Fields(definition.Title), " "))
				if strings.TrimSpace(definition.Description) != "" {
					out.Println()
					out.Println(definition.Description)
				}
				return nil
			}
			return fmt.Errorf("no documentation available for configuration key %q", key)
		},
	}
}

func defaultsUnset(config *viper.Viper) *Command {
	return &Command{
		Name:        "unset",
		Title:       "Unset one or more configuration values.",
		Description: "This command removes one or more configuration values from the configuration file completely.",
		Args: []Argument{{
			Name: "key", Repeatable: true, Prompt: "Select a configuration key", ChoicesFunc: configurationKeyChoices(config),
		}},
		RunFunc: func(_ context.Context, args *Arguments, out *OutputWriter) error {
			settings, err := getSettingsFromConfigFile(config)
			if err != nil {
				return fmt.Errorf("unable to read configuration file: %w", err)
			}

			updated := false
			for _, key := range args.GetRepeatable("key") {
				value, ok := settings[key]
				if !ok {
					out.Printf("No such configuration key: <info>%s</info>\n", key)
					continue
				}
				out.Printf("Unset <info>%s</info> (value: <info>%v</info>)\n", key, value)
				delete(settings, key)
				updated = true
			}

			if !updated {
				out.Println("Nothing to update")
				return nil
			}

			v := viper.New()
			for key, value := range settings {
				v.Set(key, value)
			}

			v.SetConfigFile(config.ConfigFileUsed())
			if err := v.WriteConfig(); err != nil {
				return fmt.Errorf("unable to save configuration file: %w", err)
			}

			out.Println("Configuration file updated")
			return nil
		},
	}
}

// ensureDirectoryExists tries to create the directory that will hold the Viper configuration file.
func ensureDirectoryExists(dir string) error {
	return os.MkdirAll(dir, 0o750)
}

// getSettingsFromConfigFile returns settings from a Viper configuration file as a map.
func getSettingsFromConfigFile(config *viper.Viper) (map[string]any, error) {
	v := viper.New()
	v.SetConfigFile(config.ConfigFileUsed())
	if err := v.ReadInConfig(); errors.Is(err, os.ErrNotExist) {
		return make(map[string]any), nil
	} else if err != nil {
		return nil, fmt.Errorf("unable to read configuration file %q: %w", config.ConfigFileUsed(), err)
	}

	return v.AllSettings(), nil
}

// configurationKeyChoices reads persisted keys for validation, interactive selection, and shell completion.
// Read the file for each request so command defaults and environment variables do not become selectable keys.
func configurationKeyChoices(config *viper.Viper) ArgumentChoicesFunc {
	return func(context.Context, *Arguments) ([]string, error) {
		settings, err := getSettingsFromConfigFile(config)
		if err != nil {
			return nil, err
		}
		return slices.Sorted(maps.Keys(settings)), nil
	}
}
