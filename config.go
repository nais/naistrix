package naistrix

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// configCommand creates the built-in config command for managing configuration.
func configCommand() *Command {
	return &Command{
		Name:  "config",
		Title: "Configuration management",
		SubCommands: []*Command{
			configSet(),
			configGet(),
			configList(),
			configUnset(),
		},
	}
}

// configSet creates the 'config set' command.
func configSet() *Command {
	return &Command{
		Name:  "set",
		Args:  []Argument{{Name: "key"}, {Name: "value"}},
		Title: "Set a configuration value",
		RunFunc: func(ctx context.Context, args *Arguments, out *OutputWriter) error {
			key := args.Get("key")
			value := args.Get("value")

			viper.Set(key, value)
			if err := viper.WriteConfig(); err != nil {
				// Config file doesn't exist, create it
				if os.IsNotExist(err) || viper.ConfigFileUsed() == "" {
					if err := viper.SafeWriteConfig(); err != nil {
						return fmt.Errorf("failed to create config: %w", err)
					}
				} else {
					return fmt.Errorf("failed to write config: %w", err)
				}
			}

			out.Printf("Set %s = %s\n", key, value)
			return nil
		},
	}
}

// configGet creates the 'config get' command.
func configGet() *Command {
	return &Command{
		Name:  "get",
		Args:  []Argument{{Name: "key"}},
		Title: "Get a configuration value",
		RunFunc: func(ctx context.Context, args *Arguments, out *OutputWriter) error {
			key := args.Get("key")

			if !viper.IsSet(key) {
				return fmt.Errorf("key %q is not set", key)
			}

			out.Printf("%s = %v\n", key, viper.Get(key))
			return nil
		},
	}
}

// configList creates the 'config list' command.
func configList() *Command {
	return &Command{
		Name:  "list",
		Title: "List all configuration values",
		RunFunc: func(ctx context.Context, args *Arguments, out *OutputWriter) error {
			settings := viper.AllSettings()
			if len(settings) == 0 {
				out.Println("No configuration values set")
				return nil
			}

			for key, value := range settings {
				out.Printf("%s = %v\n", key, value)
			}
			return nil
		},
	}
}

// configUnset creates the 'config unset' command.
func configUnset() *Command {
	return &Command{
		Name:  "unset",
		Args:  []Argument{{Name: "key"}},
		Title: "Unset a configuration value",
		RunFunc: func(ctx context.Context, args *Arguments, out *OutputWriter) error {
			key := args.Get("key")

			if !viper.IsSet(key) {
				return fmt.Errorf("key %q is not set", key)
			}

			allSettings := viper.AllSettings()
			delete(allSettings, key)

			newViper := viper.New()
			for k, v := range allSettings {
				newViper.Set(k, v)
			}

			newViper.SetConfigFile(viper.ConfigFileUsed())
			if err := newViper.WriteConfig(); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}

			out.Printf("Unset %s from configuration\n", key)
			return nil
		},
	}
}
