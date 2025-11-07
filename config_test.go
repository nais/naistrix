package naistrix_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nais/naistrix"
	"github.com/spf13/viper"
)

func TestConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	tests := []struct {
		args string
		want string
		err  string
	}{
		{
			args: "config list",
			want: "help = false",
		},
		{
			args: "config set expected_key expected_value",
			want: "Set expected_key = expected_value",
		},
		{
			args: "config get expected_key",
			want: "expected_key = expected_value",
		},
		{
			args: "config unset expected_key",
			want: "Unset expected_key from configuration\n",
		},
		{
			args: "config get expected_key",
			err:  "key \"expected_key\" is not set",
		},
	}

	for _, test := range tests {
		t.Run(test.args, func(t *testing.T) {
			got, err := runCommand(configPath, test.args)
			if test.err != "" {
				if err == nil || !strings.Contains(err.Error(), test.err) {
					t.Errorf("runCommand(%s) error = %v, want %v", test.args, err, test.err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !strings.Contains(got, test.want) {
				t.Errorf("runCommand(%s) = %v, want %+v", test.args, got, test.want)
			}
		})
	}
}

func runCommand(configPath, args string) (string, error) {
	viper.Reset()
	argSlice := []string{"--config", configPath}
	argSlice = append(argSlice, strings.Split(args, " ")...)

	var outputBuffer bytes.Buffer
	app, _, err := naistrix.NewApplication(
		"test",
		"test application",
		"v0.6.9",
		naistrix.ApplicationWithWriter(&outputBuffer),
	)
	if err != nil {
		return "", err
	}

	err = app.Run(naistrix.RunWithArgs(argSlice))
	return outputBuffer.String(), err
}
