package application

import (
	"context"

	"file-service/internal/domain"
	"github.com/ofm-microservices/ofm-common/pkg/logging"
)

// Logger aliases the shared structured logger used by the application layer.
type Logger = logging.Logger

// FileService exposes the file upload and lookup workflow owned by the
// application layer.
type FileService interface {
	CreateFile(ctx context.Context, params domain.UploadFileParams) (*domain.File, error)
	CreateFiles(ctx context.Context, params domain.UploadFilesParams) ([]*domain.File, error)
	GetFile(ctx context.Context, fileID string) (*domain.File, error)
	GetFileURL(ctx context.Context, fileID string) (string, error)
	GetFileURLs(ctx context.Context, fileIDs []string) ([]domain.FileURL, error)
	DeleteFile(ctx context.Context, fileID string) error
}
