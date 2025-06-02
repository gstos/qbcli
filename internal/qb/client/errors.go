package client

import (
	"errors"
	"fmt"
)

type RequestError struct {
	Err         error
	isTransient bool
}

func NewRequestError(isTransient bool, format string, args ...any) RequestError {
	return RequestError{Err: fmt.Errorf(format, args...), isTransient: isTransient}
}

func NewFatalError(format string, args ...any) RequestError {
	return NewRequestError(false, format, args...)
}

func NewTransientError(format string, args ...any) RequestError {
	return NewRequestError(false, format, args...)
}

func NewRequestErrorFrom(e error, isTransient bool, format string, args ...any) RequestError {
	args = append(args, e)
	return NewRequestError(isTransient, format+": %w", args...)
}

func FatalErrorFrom(e error, format string, args ...any) RequestError {
	return NewRequestErrorFrom(e, false, format, args...)
}

func TransientErrorFrom(e error, format string, args ...any) RequestError {
	return NewRequestErrorFrom(e, true, format, args...)
}

func WrapFatalUnlessExplicit(e error, format string, args ...any) error {
	if reqErr, ok := IsRequestError(e); !ok {
		return NewRequestErrorFrom(reqErr.Err, reqErr.isTransient, format, args...)
	}
	return FatalErrorFrom(e, format, args...)
}

func IsTransientError(e error) bool {
	// All errors are fatal, except the ones marked as transient
	if reqErr, ok := IsRequestError(e); ok {
		return reqErr.isTransient
	}
	return false
}

func IsRequestError(e error) (RequestError, bool) {
	var reqErr RequestError
	if errors.As(e, &reqErr) {
		return reqErr, true
	}
	return reqErr, false
}

func (e RequestError) Error() string     { return e.Err.Error() }
func (e RequestError) Unwrap() error     { return e.Err }
func (e RequestError) IsTransient() bool { return e.isTransient }
func (e RequestError) IsFatal() bool     { return !e.isTransient }
