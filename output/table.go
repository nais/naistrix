package output

import (
	"fmt"
	"io"
	"reflect"

	"github.com/pterm/pterm"
)

// TableOptionFunc is a function that can be used to configure a Table.
type TableOptionFunc func(*Table)

// TableWithShowHiddenColumns can be used to force rendering all exported fields in a struct, even if the field have the
// `hidden:"true"` tag.
func TableWithShowHiddenColumns() TableOptionFunc {
	return func(t *Table) {
		t.showHidden = true
	}
}

type Table struct {
	showHidden   bool
	tablePrinter *pterm.TablePrinter
}

// NewTable creates a new Table that will write to the provided io.Writer. The table can be configured using the
// available TableOptionFunc functions.
func NewTable(w io.Writer, opts ...TableOptionFunc) *Table {
	t := &Table{
		tablePrinter: pterm.DefaultTable.WithWriter(w),
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// Render will render the table with the passed data. The data needs to be a slice of structs, and all exported fields
// in the struct will be added as columns. The field names will be used as headers, and can be overridden using a
// `heading` field tag. Fields can be hidden using a `hidden` field tag set to "true". To show hidden fields, use the
// TableWithShowHiddenColumns option when creating the table.
func (t *Table) Render(v any) error {
	tableData, err := t.convert(v)
	if err != nil {
		return err
	}

	return t.tablePrinter.
		WithHasHeader(true).
		WithHeaderRowSeparator("-").
		WithData(tableData).
		Render()
}

// convert converts the provided data into pterm.TableData. The data must be a slice of structs, and the struct must
// have at least one exported field.
func (t *Table) convert(v any) (pterm.TableData, error) {
	d := reflect.ValueOf(v)

	if d.Kind() != reflect.Slice {
		return nil, fmt.Errorf("data must be a slice, got %T", v)
	}

	if d.Len() == 0 {
		return nil, fmt.Errorf("data slice is empty")
	}

	// extract headers from the first struct in the slice
	headers, err := t.extractHeaders(d.Index(0))
	if err != nil {
		return nil, err
	}

	if len(headers) == 0 {
		return nil, fmt.Errorf("no visible fields in struct")
	}

	td := pterm.TableData{headers}
	for i := 0; i < d.Len(); i++ {
		row := d.Index(i)

		if row.Kind() == reflect.Pointer {
			if row.IsNil() {
				return nil, fmt.Errorf("nil pointer in slice at index %d", i)
			}
			row = row.Elem()
		}

		td = append(td, columnsInRow(row, t.showHidden))
	}

	return td, nil
}

// extractHeaders returns a slice of header strings extracted from the struct fields of the provided value.
func (t *Table) extractHeaders(v reflect.Value) ([]string, error) {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, fmt.Errorf("nil pointer in sice at index 0")
		}

		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("value must be a struct, got %T", v.Interface())
	}

	headers := make([]string, 0)
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		if !field.IsExported() {
			continue
		}

		if field.Tag.Get("hidden") == "true" && !t.showHidden {
			continue
		}

		heading := field.Name
		if tag := field.Tag.Get("heading"); tag != "" {
			heading = tag
		}
		headers = append(headers, heading)
	}

	if len(headers) == 0 {
		return nil, fmt.Errorf("no visible fields in struct")
	}

	return headers, nil
}

// columnsInRow returns a slice of strings representing the values of the exported fields (the columns) in the provided
// struct value (the row).
func columnsInRow(row reflect.Value, showHidden bool) []string {
	fields := reflect.TypeOf(row.Interface())
	values := reflect.ValueOf(row.Interface())

	cols := make([]string, 0)
	for i := range fields.NumField() {
		if !fields.Field(i).IsExported() {
			continue
		}

		if fields.Field(i).Tag.Get("hidden") == "true" && !showHidden {
			continue
		}

		cols = append(cols, getStringValue(values.Field(i)))
	}

	return cols
}

// getStringValue returns the string representation of the provided reflect.Value.
func getStringValue(v reflect.Value) string {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if !v.IsValid() {
		return ""
	}

	return fmt.Sprint(v.Interface())
}
