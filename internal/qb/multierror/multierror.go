package multierror

import (
	"fmt"
	"strings"
)

type MultiError struct {
	Errors []error
}

func (m MultiError) Error() string {
	var b strings.Builder
	_, fmtErr := fmt.Fprintf(&b, "%d GraphQL error(s):\n", len(m.Errors))
	if fmtErr != nil {
		return fmt.Sprintf("failed to format query errors: %v", fmtErr)
	}
	for i, err := range m.Errors {
		_, fmtErr := fmt.Fprintf(&b, "  %d. %s\n", i+1, err)
		if fmtErr != nil {
			return fmt.Sprintf("failed to format query errors: %v", fmtErr)
		}
	}
	return b.String()
}

func (m MultiError) Add(err error) {
	m.Errors = append(m.Errors, err)
}

func (m MultiError) Unwrap() error {
	if len(m.Errors) == 0 {
		return nil
	}
	return m
}

func NewMultiError(errors ...error) MultiError {
	multiError := MultiError{Errors: errors}
	return multiError
}

func WrapIfError(errs []error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return NewMultiError(errs...)
	}
}
