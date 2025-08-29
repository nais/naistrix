// Package output defines an interface for rendering data, along with some implementations. This is used for more
// complex output formats like for instance tables and JSON.
package output

type Renderer interface {
	Render(v any) error
}
