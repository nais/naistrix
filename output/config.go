package output

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

// Config is a renderer that encodes values as Config and writes them to an [io.Writer]. Use [NewConfig] to construct one.
type Config struct {
	writer io.Writer
}

// NewConfig creates a new [Config] renderer that will write to the provided [io.Writer].
func NewConfig(w io.Writer) *Config {
	return &Config{
		writer: w,
	}
}

// Render encodes v as Config and writes the result to the configured [io.Writer].
func (c *Config) Render(data map[string]any) error {
	if len(data) == 0 {
		return fmt.Errorf("data must be a non-empty map")
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	var builder strings.Builder
	for _, key := range keys {
		fmt.Fprintf(&builder, "%s=%s\n", key, data[key])
	}

	_, err := c.writer.Write([]byte(builder.String()))
	return err
}
