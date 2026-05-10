package grpc

import "errors"

var (
	// ErrNilFileService reports a missing application service dependency.
	ErrNilFileService = errors.New("file service is nil")
	// ErrNilLogger reports a missing logger dependency.
	ErrNilLogger = errors.New("logger is nil")
)
