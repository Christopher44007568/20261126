package dds

import (
	"fmt"
	"strings"
)

// FieldError identifies the DDS field or data range that failed validation.
// Offset is relative to the start of the DDS file and is -1 when not applicable.
type FieldError struct {
	Field    string `json:"field"`
	Offset   int64  `json:"offset,omitempty"`
	Message  string `json:"message"`
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
}

func (e FieldError) Error() string {
	var b strings.Builder
	if e.Field != "" {
		b.WriteString(e.Field)
		b.WriteString(": ")
	}
	b.WriteString(e.Message)
	if e.Expected != "" || e.Actual != "" {
		fmt.Fprintf(&b, " (expected %s, got %s)", e.Expected, e.Actual)
	}
	return b.String()
}

// DecodeError carries every independently detectable field error. Parsing
// stops before allocating a Texture when any field is invalid.
type DecodeError struct {
	Errors []FieldError `json:"errors"`
}

func (e *DecodeError) Error() string {
	if len(e.Errors) == 0 {
		return "invalid DDS"
	}
	parts := make([]string, len(e.Errors))
	for i := range e.Errors {
		parts[i] = e.Errors[i].Error()
	}
	return strings.Join(parts, "; ")
}

func (e *DecodeError) add(field, message string) {
	e.Errors = append(e.Errors, FieldError{Field: field, Offset: -1, Message: message})
}

func (e *DecodeError) addAt(field string, offset int64, message, expected, actual string) {
	e.Errors = append(e.Errors, FieldError{
		Field: field, Offset: offset, Message: message, Expected: expected, Actual: actual,
	})
}
