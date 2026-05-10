package scylla

const (
	createFileQuery = `
		INSERT INTO files (
			file_id, owner_id, filename, extension, content_type, bucket, storage_path, size_bytes, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	getFileByIDQuery = `
		SELECT file_id, owner_id, filename, extension, content_type, bucket, storage_path, size_bytes, created_at, updated_at
		FROM files
		WHERE file_id = ?
	`
	deleteFileByIDQuery = `DELETE FROM files WHERE file_id = ?`
)
