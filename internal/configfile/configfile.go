// Package configfile handles configuration file paths, migration, and persisted settings.
package configfile

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/viper"
)

// ResolveHomeDir resolves the home directory in path if it starts with "~/".
func ResolveHomeDir(path string) (string, error) {
	if len(path) > 1 && path[:2] == "~/" {
		u, err := user.Current()
		if err != nil {
			return "", err
		}
		path = filepath.Join(u.HomeDir, path[2:])
	}
	return path, nil
}

// MigrateLegacy moves filename from oldDir to configDir and removes oldDir after a successful move.
// Migration is best-effort: missing legacy files and other errors are ignored.
func MigrateLegacy(configDir, oldDir, filename string) {
	stat, err := os.Stat(oldDir)
	if err != nil || !stat.IsDir() {
		return
	}
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		return
	}
	if err := os.Rename(filepath.Join(oldDir, filename), filepath.Join(configDir, filename)); err != nil {
		return
	}
	_ = os.RemoveAll(oldDir)
}

// ReadSettings reads persisted settings without environment overrides or flag defaults.
// A missing file returns an empty map. Each call reads the file again using a fresh Viper instance.
func ReadSettings(path string) (map[string]any, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); errors.Is(err, os.ErrNotExist) {
		return make(map[string]any), nil
	} else if err != nil {
		return nil, fmt.Errorf("unable to read configuration file %q: %w", path, err)
	}
	return v.AllSettings(), nil
}
