package model

import "time"

// FileRow mirrors the authoritative file metadata stored in ScyllaDB.
type FileRow struct {
	ID          string    `db:"file_id"`
	OwnerID     string    `db:"owner_id"`
	Filename    string    `db:"filename"`
	Extension   string    `db:"extension"`
	ContentType string    `db:"content_type"`
	Bucket      string    `db:"bucket"`
	StoragePath string    `db:"storage_path"`
	SizeBytes   int64     `db:"size_bytes"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}
