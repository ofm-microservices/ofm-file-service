package domain

import "errors"

var (
	// ErrInvalidFileID reports an empty or malformed file identifier.
	ErrInvalidFileID = errors.New("invalid file id")
	// ErrInvalidOwnerID reports an empty or malformed owner identifier.
	ErrInvalidOwnerID = errors.New("invalid owner id")
	// ErrInvalidFilename reports an empty filename.
	ErrInvalidFilename = errors.New("invalid filename")
	// ErrInvalidContentType reports an empty file content type.
	ErrInvalidContentType = errors.New("invalid content type")
	// ErrInvalidStoragePath reports an empty storage path.
	ErrInvalidStoragePath = errors.New("invalid storage path")
	// ErrInvalidFileData reports an empty file body.
	ErrInvalidFileData = errors.New("invalid file data")
	// ErrFileNotFound reports that the file metadata does not exist.
	ErrFileNotFound = errors.New("file not found")
	// ErrFailedToCreateFile reports a repository create failure.
	ErrFailedToCreateFile = errors.New("failed to create file")
	// ErrFailedToFindFile reports a repository lookup failure.
	ErrFailedToFindFile = errors.New("failed to find file")
	// ErrFailedToDeleteFile reports a repository delete failure.
	ErrFailedToDeleteFile = errors.New("failed to delete file")
	// ErrFailedToStoreFile reports an object storage failure.
	ErrFailedToStoreFile = errors.New("failed to store file")
	// ErrFailedToDeleteStoredFile reports an object removal failure.
	ErrFailedToDeleteStoredFile = errors.New("failed to delete stored file")
)
