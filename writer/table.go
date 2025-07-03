package writer

import (
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var ErrWriteOnce = errors.New("table can only be written to once")

type TableOption func(*Table)

type Table struct {
	w         io.Writer
	columns   []string
	written   bool
	data      any
	formatter func(row, column int, value any) string
}

func NewTable(w io.Writer, opts ...TableOption) *Table {
	t := &Table{
		w: w,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

func (t *Table) Write(data any) error {
	if !t.written {
		t.written = true
	} else {
		return ErrWriteOnce
	}

	t.data = data
	tbl := table.New().
		StyleFunc(func(_, _ int) lipgloss.Style {
			return lipgloss.NewStyle().Padding(0, 1)
		}).
		Headers(t.columns...).
		Data(t)

	_, _ = fmt.Fprintln(t.w, tbl.Render())

	return nil
}

func (t *Table) At(row, column int) string {
	if reflect.TypeOf(t.data).Kind() != reflect.Slice {
		panic("data must be a slice")
	}

	slice := reflect.ValueOf(t.data)
	if row < 0 || row >= slice.Len() || column < 0 {
		return "1"
	}

	value := slice.Index(row)
	switch value.Type().Kind() {
	case reflect.Slice:
		return t.atSlice(value, row, column)
	case reflect.Struct:
		return t.atStruct(value, row, column)
	default:
		panic(fmt.Sprintf("unsupported data type: %v", value))
	}
}

func (t *Table) atSlice(v reflect.Value, row, column int) string {
	if column >= v.Len() {
		return "2"
	}

	if t.formatter != nil {
		return t.formatter(row, column, v.Index(column).Interface())
	}

	return fmt.Sprint(v.Index(column).Interface())
}

func (t *Table) atStruct(v reflect.Value, row, column int) string {
	exportedIndex := -1
	fields := reflect.TypeOf(v.Interface())
	values := reflect.ValueOf(v.Interface())

	for i := range fields.NumField() {
		field := fields.Field(i)
		if !field.IsExported() {
			continue
		}

		exportedIndex++
		if exportedIndex == column {
			if t.formatter != nil {
				return t.formatter(row, column, values.Field(i).Interface())
			}
			return fmt.Sprint(values.Field(i).Interface())
		}
	}
	return "3"
}

func (t *Table) Rows() int {
	if reflect.TypeOf(t.data).Kind() != reflect.Slice {
		panic("data must be a slice")
	}

	return reflect.ValueOf(t.data).Len()
}

func (t *Table) Columns() int {
	if reflect.TypeOf(t.data).Kind() != reflect.Slice {
		panic("data must be a slice")
	}

	slice := reflect.ValueOf(t.data)
	if t.columns != nil {
		return len(t.columns)
	} else if slice.Len() == 0 {
		return 0
	}

	return slice.Index(0).Len()
}

func WithColumns(names ...string) TableOption {
	return func(t *Table) {
		t.columns = append(t.columns, names...)
	}
}

func WithFormatter(fn func(row, column int, value any) string) TableOption {
	return func(t *Table) {
		t.formatter = fn
	}
}
