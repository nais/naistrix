package configfile_test

import (
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/nais/naistrix/internal/configfile"
)

func TestResolveHomeDir(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	absolute := filepath.Join(t.TempDir(), "config.yaml")
	for _, tt := range []struct {
		path string
		want string
	}{
		{path: "", want: ""},
		{path: "config.yaml", want: "config.yaml"},
		{path: absolute, want: absolute},
		{path: "~", want: "~"},
		{path: "~someone/config.yaml", want: "~someone/config.yaml"},
		{path: "~/config.yaml", want: filepath.Join(u.HomeDir, "config.yaml")},
		{path: "~/.config/example/config.yaml", want: filepath.Join(u.HomeDir, ".config", "example", "config.yaml")},
	} {
		t.Run(tt.path, func(t *testing.T) {
			if got, err := configfile.ResolveHomeDir(tt.path); err != nil || got != tt.want {
				t.Errorf("ResolveHomeDir(%q) = %q, %v; want %q", tt.path, got, err, tt.want)
			}
		})
	}
}

func TestMigrateLegacy(t *testing.T) {
	for _, mode := range []string{"success", "missing directory", "not a directory", "missing file", "blocked destination", "rename fails"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			oldDir := filepath.Join(root, ".example")
			newDir := filepath.Join(root, "example")
			const contents = "team: my-team\n"
			if mode == "not a directory" {
				if err := os.WriteFile(oldDir, []byte(contents), 0o600); err != nil {
					t.Fatal(err)
				}
			} else if mode != "missing directory" {
				if err := os.Mkdir(oldDir, 0o750); err != nil {
					t.Fatal(err)
				}
				if mode != "missing file" {
					if err := os.WriteFile(filepath.Join(oldDir, "config.yaml"), []byte(contents), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			}
			switch mode {
			case "blocked destination":
				if err := os.WriteFile(newDir, []byte("not a directory"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "rename fails":
				if err := os.MkdirAll(filepath.Join(newDir, "config.yaml"), 0o750); err != nil {
					t.Fatal(err)
				}
			}

			configfile.MigrateLegacy(newDir, oldDir, "config.yaml")
			if mode == "success" {
				got, err := os.ReadFile(filepath.Join(newDir, "config.yaml"))
				if err != nil || string(got) != contents {
					t.Fatalf("expected migrated configuration, got %q, %v", got, err)
				}
				if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
					t.Errorf("expected legacy directory to be removed, got %v", err)
				}
			} else if mode != "missing directory" {
				if _, err := os.Stat(oldDir); err != nil {
					t.Fatalf("failed migration removed legacy path: %v", err)
				}
				if mode == "blocked destination" || mode == "rename fails" {
					got, err := os.ReadFile(filepath.Join(oldDir, "config.yaml"))
					if err != nil || string(got) != contents {
						t.Errorf("failed migration modified legacy configuration: %q, %v", got, err)
					}
				}
			}
		})
	}
}

func TestReadSettings(t *testing.T) {
	t.Setenv("TEAM", "environment-team")
	for _, tt := range []struct {
		name     string
		contents string
		want     map[string]any
	}{
		{name: "missing", want: map[string]any{}},
		{name: "empty", contents: "", want: map[string]any{}},
		{name: "empty map", contents: "{}\n", want: map[string]any{}},
		{name: "values", contents: "TEAM: my-team\nconfirm-login: false\nempty: \"\"\n", want: map[string]any{"team": "my-team", "confirm-login": false, "empty": ""}},
		{name: "malformed", contents: "team: [invalid yaml\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if tt.name != "missing" {
				if err := os.WriteFile(path, []byte(tt.contents), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			got, err := configfile.ReadSettings(path)
			if tt.name == "malformed" {
				if err == nil || !strings.HasPrefix(err.Error(), "unable to read configuration file \""+path+"\"") {
					t.Fatalf("expected error with configuration path, got %v", err)
				}
			} else if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ReadSettings() = %v, %v; want %v", got, err, tt.want)
			}
			if tt.name == "missing" {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Errorf("ReadSettings created configuration file: %v", err)
				}
			} else if saved, err := os.ReadFile(path); err != nil || string(saved) != tt.contents {
				t.Errorf("ReadSettings modified configuration: %q, %v", saved, err)
			}
			if tt.name == "values" {
				if err := os.WriteFile(path, []byte("team: another-team\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				got, err := configfile.ReadSettings(path)
				if err != nil || !reflect.DeepEqual(got, map[string]any{"team": "another-team"}) {
					t.Errorf("ReadSettings returned stale settings: %v, %v", got, err)
				}
			}
		})
	}
}
