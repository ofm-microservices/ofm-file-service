package postgres

import (
	"errors"
	"fmt"
)

var ErrNilPostgresDB = errors.New("postgres database is nil")
var ErrNilLogger = errors.New("logger is nil")

func WrapCreateFileError(err error) error { return fmt.Errorf("create file: %w", err) }
func WrapFindFileError(err error) error   { return fmt.Errorf("find file: %w", err) }
func WrapDeleteFileError(err error) error { return fmt.Errorf("delete file: %w", err) }
