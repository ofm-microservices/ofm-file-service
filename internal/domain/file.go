package domain

import (
	"time"
)

// File describes one uploaded object and its metadata.
type File struct {
	ID          string
	OwnerID     string
	Filename    string
	Extension   string
	ContentType string
	Bucket      string
	StoragePath string
	SizeBytes   int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// FileURL pairs a file identifier with a public URL for read-model and API
// responses.
type FileURL struct {
	ID  string
	URL string
}
