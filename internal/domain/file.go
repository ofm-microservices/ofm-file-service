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
