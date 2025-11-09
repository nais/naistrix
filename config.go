package naistrix

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/viper"
	"golang.org/x/exp/maps"
)

// configCommand creates the built-in config command for managing configuration.
func configCommand(config *viper.Viper) *Command {
	return &Command{
		Name:  "config",
		Title: "Manage configuration file / values.",
		Description: heredoc.Docf(`
			The config command allows you to set, get, unset and list configuration values stored in the configuration file.

			Configuration values acts as defaults for various flags throughout the application.
		`),
		SubCommands: []*Command{
			configSet(config),
			configGet(config),
			configList(config),
			configUnset(config),
		},
	}
}

func configSet(config *viper.Viper) *Command {
	return &Command{
		Name: "set",
		Args: []Argument{
			{Name: "key"},
			{Name: "value"},
		},
		Title:       "Set a configuration value",
		Description: "Set a configuration value in the configuration file. This value will be used as default for relevant flags throughout the application.",
		AutoCompleteFunc: func(_ context.Context, args *Arguments, _ string) ([]string, string) {
			settings, err := getSettingsFromConfigFile(config.ConfigFileUsed())
			if err != nil {
				return []string{}, ""
			}

			return maps.Keys(settings), "Choose an existing key or create a new one"
		},
		RunFunc: func(_ context.Context, args *Arguments, out *OutputWriter) error {
			configFilePath := config.ConfigFileUsed()
			dir := filepath.Dir(configFilePath)

			if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
				if ok, err := out.Confirm("The directory for the configuration file (%s) does not exist, do you want to create it?", dir); err != nil {
					return err
				} else if !ok {
					out.Warnln("Directory creation aborted; configuration not saved")
					return nil
				}
			} else if err != nil {
				return fmt.Errorf("unable to access directory %q for configuration file: %w", dir, err)
			}

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

func configGet(config *viper.Viper) *Command {
	return &Command{
		Name:             "get",
		Title:            "Get one or more configuration values.",
		Description:      "This command retrieves one or more configuration values from the configuration file.",
		Args:             []Argument{{Name: "key", Repeatable: true}},
		AutoCompleteFunc: autoCompleteConfigurationKeys(config.ConfigFileUsed()),
		RunFunc: func(_ context.Context, args *Arguments, out *OutputWriter) error {
			settings, err := getSettingsFromConfigFile(config.ConfigFileUsed())
			if err != nil {
				return fmt.Errorf("unable to read configuration file: %w", err)
			}

			for _, key := range args.GetRepeatable("key") {
				value, ok := settings[key]
				if !ok {
					out.Printf("No such configuration key: <info>%s</info>, create the value using <info>config set %s <value></info>\n", key, key)
					continue
				}

				out.Printf("<info>%s</info> = <info>%v</info>\n", key, value)

			}
			return nil
		},
	}
}

func configList(config *viper.Viper) *Command {
	return &Command{
		Name:  "list",
		Title: "List all configuration values found in the configuration file.",
		RunFunc: func(_ context.Context, _ *Arguments, out *OutputWriter) error {
			settings, err := getSettingsFromConfigFile(config.ConfigFileUsed())
			if err != nil {
				return fmt.Errorf("unable to read configuration file: %w", err)
			}

			if len(settings) == 0 {
				out.Printf("The configuration file <info>%s</info> is empty, or it does not yet exist\n", config.ConfigFileUsed())
				out.Println("Use the <info>config set <key> <value></info> command to set configuration values")
				return nil
			}

			values := make([][]string, 0)
			for k, v := range settings {
				values = append(values, []string{k, fmt.Sprint(v)})
			}

			sort.SliceStable(values, func(i, j int) bool {
				if len(values[i]) == 0 || len(values[j]) == 0 {
					return false
				}
				return values[i][0] < values[j][0]
			})

			values = append([][]string{{"Key", "Value"}}, values...)
			out.Printf("The following configuration values are set in <info>%s</info>:\n\n", config.ConfigFileUsed())
			_ = out.Table().Render(values)
			out.Println("\nUse the <info>config set <key> <value></info> command to update or create values, or the <info>config unset <value>[, <value>]</info> command to remove values")
			return nil
		},
	}
}

func configUnset(config *viper.Viper) *Command {
	return &Command{
		Name:             "unset",
		Title:            "Unset one or more configuration values.",
		Description:      "This command removes one or more configuration values from the configuration file completely.",
		Args:             []Argument{{Name: "key", Repeatable: true}},
		AutoCompleteFunc: autoCompleteConfigurationKeys(config.ConfigFileUsed()),
		RunFunc: func(_ context.Context, args *Arguments, out *OutputWriter) error {
			settings, err := getSettingsFromConfigFile(config.ConfigFileUsed())
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
func getSettingsFromConfigFile(path string) (map[string]any, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); errors.Is(err, os.ErrNotExist) {
		return make(map[string]any), nil
	} else if err != nil {
		return nil, fmt.Errorf("unable to read configuration file %q: %w", path, err)
	}

	return v.AllSettings(), nil
}

// autoCompleteConfigurationKeys returns an AutoCompleteFunc that suggests configuration keys from the given config
// file.
func autoCompleteConfigurationKeys(configFile string) AutoCompleteFunc {
	settings, err := getSettingsFromConfigFile(configFile)
	if err != nil {
		return nil
	}

	return func(_ context.Context, args *Arguments, _ string) ([]string, string) {
		var inArgs []string
		if args.Len() > 0 {
			inArgs = args.GetRepeatable("key")
		}

		keys := make([]string, 0)
		for key := range settings {
			if slices.Contains(inArgs, key) {
				continue
			}

			keys = append(keys, key)
		}

		if len(keys) == 0 {
			return []string{}, ""
		}

		return keys, "Available configuration keys"
	}
}
