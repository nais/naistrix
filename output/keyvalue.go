package output

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

// KeyValue is a renderer that encodes values as KeyValue and writes them to an [io.Writer]. Use [NewKeyValue] to construct one.
type KeyValue struct {
	writer io.Writer
}

// NewKeyValue creates a new [KeyValue] renderer that will write to the provided [io.Writer].
func NewKeyValue(w io.Writer) *KeyValue {
	return &KeyValue{
		writer: w,
	}
}

// Render encodes data as key=value and writes the result to the configured [io.Writer].
// Output is unquoted to allow unescaped JSON, so values with '=' needs to be quoted by the user.
func (kv *KeyValue) Render(data map[string]any) error {
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

	_, err := kv.writer.Write([]byte(builder.String()))
	return err
}
