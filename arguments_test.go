package naistrix

import (
	"slices"
	"testing"
)

func TestInput_All(t *testing.T) {
	t.Run("no arguments", func(t *testing.T) {
		if got := newArguments([]Argument{}, []string{}).All(); len(got) > 0 {
			t.Errorf("expected no arguments, got: %v", got)
		}
	})

	t.Run("regular arguments", func(t *testing.T) {
		cobraArgs := []string{"v1", "v2"}
		args := newArguments([]Argument{{Name: "a1"}, {Name: "a2"}}, cobraArgs)
		if args.Len() != len(cobraArgs) {
			t.Errorf("expected %d arguments, got: %d", len(cobraArgs), args.Len())
		}

		got := args.All()
		if expected := cobraArgs; !slices.Equal(got, expected) {
			t.Errorf(`expected args to be %q, got: %q`, expected, got)
		}
	})

	t.Run("with repeatable", func(t *testing.T) {
		cobraArgs := []string{"v1", "v2", "v3", "v4"}
		args := newArguments([]Argument{{Name: "a1"}, {Name: "a2", Repeatable: true}}, cobraArgs)
		got := args.All()
		if expected := cobraArgs; !slices.Equal(got, expected) {
			t.Errorf(`expected args to be %q, got: %q`, expected, got)
		}
	})

	t.Run("only repeatable", func(t *testing.T) {
		cobraArgs := []string{"v1", "v2", "v3", "v4"}
		args := newArguments([]Argument{{Name: "a1", Repeatable: true}}, cobraArgs)
		got := args.All()
		if expected := cobraArgs; !slices.Equal(got, expected) {
			t.Errorf(`expected args to be %q, got: %q`, expected, got)
		}
	})
}

func TestArgumentsLen(t *testing.T) {
	for _, tt := range []struct {
		name    string
		defined []Argument
		values  []string
	}{
		{name: "no values"},
		{name: "missing value", defined: []Argument{{Name: "first"}, {Name: "second"}}, values: []string{"one"}},
		{name: "extra value", defined: []Argument{{Name: "first"}}, values: []string{"one", "two"}},
		{name: "repeatable values", defined: []Argument{{Name: "first"}, {Name: "rest", Repeatable: true}}, values: []string{"one", "two", "three"}},
		{name: "only repeatable values", defined: []Argument{{Name: "values", Repeatable: true}}, values: []string{"one", "two"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			args := newArguments(tt.defined, tt.values)
			if got, want := args.Len(), len(tt.values); got != want {
				t.Errorf("Len() = %d, want %d", got, want)
			}
			if err := ValidateExactArgs(len(tt.values))(t.Context(), args); err != nil {
				t.Errorf("exact count validation failed: %v", err)
			}
		})
	}

	t.Run("extra values fail exact validation", func(t *testing.T) {
		args := newArguments([]Argument{{Name: "first"}}, []string{"one", "two"})
		if err := ValidateExactArgs(1)(t.Context(), args); err == nil {
			t.Error("expected extra values to fail exact count validation")
		}
	})
}

func TestInput_Get(t *testing.T) {
	cobraArgs := []string{"v1", "v2", "v3", "v4"}
	args := newArguments(
		[]Argument{
			{Name: "a1"},
			{Name: "a2", Repeatable: true},
		},
		cobraArgs,
	)

	t.Run("get regular arg", func(t *testing.T) {
		got := args.Get("a1")
		if expected := cobraArgs[0]; got != expected {
			t.Errorf(`expected argument "a1" to be %q, got: %q`, expected, got)
		}
	})

	t.Run("get repeatable arg", func(t *testing.T) {
		got := args.GetRepeatable("a2")
		if expected := cobraArgs[1:]; !slices.Equal(got, expected) {
			t.Errorf(`expected "a2" to be %q, got: %q`, expected, got)
		}
	})

	t.Run("get regular arg as repeatable", func(t *testing.T) {
		defer func() {
			expectedError := `"a1" is not a valid repeatable argument`
			if r := recover(); r == nil {
				t.Errorf("expected panic, but function did not panic")
			} else if r != expectedError {
				t.Errorf(`expected panic with %q, got: %q`, expectedError, r)
			}
		}()

		args.GetRepeatable("a1")
	})

	t.Run("get repeatable arg as regular", func(t *testing.T) {
		defer func() {
			expectedError := `"a2" is not a valid argument`
			if r := recover(); r == nil {
				t.Errorf("expected panic, but function did not panic")
			} else if r != expectedError {
				t.Errorf(`expected panic with %q, got: %q`, expectedError, r)
			}
		}()

		args.Get("a2")
	})

	t.Run("get non-existing regular arg", func(t *testing.T) {
		defer func() {
			expectedError := `"foo" is not a valid argument`
			if r := recover(); r == nil {
				t.Errorf("expected panic, but function did not panic")
			} else if r != expectedError {
				t.Errorf(`expected panic with %q, got: %q`, expectedError, r)
			}
		}()

		args.Get("foo")
	})

	t.Run("get non-existing repeatable arg", func(t *testing.T) {
		defer func() {
			expectedError := `"foo" is not a valid repeatable argument`
			if r := recover(); r == nil {
				t.Errorf("expected panic, but function did not panic")
			} else if r != expectedError {
				t.Errorf(`expected panic with %q, got: %q`, expectedError, r)
			}
		}()

		args.GetRepeatable("foo")
	})
}
