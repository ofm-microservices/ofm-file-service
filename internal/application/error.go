package application

import "errors"

var (
	// ErrNilFileRepository reports a missing write-model repository dependency.
	ErrNilFileRepository = errors.New("file repository is nil")
	// ErrNilFileStorage reports a missing object storage dependency.
	ErrNilFileStorage = errors.New("file storage is nil")
	// ErrEmptyBucket reports a missing RustFS bucket name.
	ErrEmptyBucket = errors.New("bucket is empty")
	// ErrNilLogger reports a missing logger dependency.
	ErrNilLogger = errors.New("logger is nil")
)
