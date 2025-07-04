// Package writer defines an interface for writing data.
package writer

type Writer interface {
	Write(v any) error
}
