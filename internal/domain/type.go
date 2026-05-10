package domain

import "context"

// UploadFileParams carries the request data needed to store a file object.
type UploadFileParams struct {
	OwnerID     string
	Filename    string
	ContentType string
	Prefix      string
	Data        []byte
}

// UploadFilesParams carries a batch of file objects that should be stored in
// one request.
type UploadFilesParams struct {
	OwnerID string
	Prefix  string
	Files   []UploadFileParams
}

// FileRepository persists the authoritative file metadata write model.
type FileRepository interface {
	Create(ctx context.Context, file File) (*File, error)
	GetByID(ctx context.Context, fileID string) (*File, error)
	DeleteByID(ctx context.Context, fileID string) error
}

// FileStorage persists and deletes file objects in RustFS.
type FileStorage interface {
	Put(ctx context.Context, objectKey, contentType string, data []byte) (int64, error)
	Delete(ctx context.Context, objectKey string) error
}
