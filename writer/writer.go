package writer

type Writer interface {
	Write(v any) error
}
