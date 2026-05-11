package scylla

import (
	"errors"
	"fmt"
)

// ErrNilScyllaDB reports a missing ScyllaDB dependency.
var ErrNilScyllaDB = fmt.Errorf("scylla db is nil")
var ErrNilLogger = errors.New("logger is nil")

// WrapCreateFileError annotates file creation failures.
func WrapCreateFileError(err error) error {
	return fmt.Errorf("create file: %w", err)
}

// WrapFindFileError annotates file lookup failures.
func WrapFindFileError(err error) error {
	return fmt.Errorf("find file: %w", err)
}

// WrapDeleteFileError annotates file delete failures.
func WrapDeleteFileError(err error) error {
	return fmt.Errorf("delete file: %w", err)
}
