package helpers

import "fmt"

type FormatError struct {
	Operation string
	Details   string
	Err       error
}

func (e *FormatError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Operation, e.Details, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Operation, e.Details)
}

func WrapError(err error, operation, details string) error {
	if err == nil {
		return nil
	}
	return &FormatError{Operation: operation, Details: details, Err: err}
}
