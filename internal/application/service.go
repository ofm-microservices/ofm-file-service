package application

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"file-service/internal/domain"
	"github.com/google/uuid"
	"github.com/ofm-microservices/ofm-common/pkg/logging"
)

type fileService struct {
	repo    domain.FileRepository
	storage domain.FileStorage
	bucket  string
	log     Logger
}

// New constructs the file application service.
func New(repo domain.FileRepository, storage domain.FileStorage, bucket string, log Logger) (FileService, error) {
	if repo == nil {
		return nil, ErrNilFileRepository
	}
	if storage == nil {
		return nil, ErrNilFileStorage
	}
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return nil, ErrEmptyBucket
	}
	if log == nil {
		return nil, ErrNilLogger
	}

	return &fileService{
		repo:    repo,
		storage: storage,
		bucket:  bucket,
		log:     log.With(logging.String("module", "application")),
	}, nil
}

func (s *fileService) CreateFile(ctx context.Context, params domain.UploadFileParams) (*domain.File, error) {
	files, err := s.CreateFiles(ctx, domain.UploadFilesParams{
		OwnerID: params.OwnerID,
		Prefix:  params.Prefix,
		Files:   []domain.UploadFileParams{params},
	})
	if err != nil {
		return nil, err
	}

	return files[0], nil
}

func (s *fileService) CreateFiles(ctx context.Context, params domain.UploadFilesParams) ([]*domain.File, error) {
	ownerID := strings.TrimSpace(params.OwnerID)
	prefix := strings.Trim(strings.TrimSpace(params.Prefix), "/")
	if ownerID == "" {
		return nil, domain.ErrInvalidOwnerID
	}
	if prefix == "" {
		return nil, domain.ErrInvalidStoragePath
	}
	if len(params.Files) == 0 {
		return nil, domain.ErrInvalidFileData
	}

	created := make([]*domain.File, 0, len(params.Files))
	for _, item := range params.Files {
		file, err := s.createOne(ctx, ownerID, prefix, item)
		if err != nil {
			s.compensateCreatedFiles(created)
			return nil, err
		}
		created = append(created, file)
	}

	return created, nil
}

func (s *fileService) createOne(ctx context.Context, ownerID, prefix string, params domain.UploadFileParams) (*domain.File, error) {
	filename := strings.TrimSpace(params.Filename)
	contentType := strings.TrimSpace(params.ContentType)
	switch {
	case filename == "":
		return nil, domain.ErrInvalidFilename
	case contentType == "":
		return nil, domain.ErrInvalidContentType
	case len(params.Data) == 0:
		return nil, domain.ErrInvalidFileData
	}

	fileID := uuid.Must(uuid.NewV7()).String()
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(filename)), ".")
	objectKey := buildObjectKey(prefix, fileID, ext)

	size, err := s.storage.Put(ctx, objectKey, contentType, params.Data)
	if err != nil {
		s.log.Error("store file failed", logging.String("file_id", fileID), logging.Err(err))
		return nil, domain.ErrFailedToStoreFile
	}

	now := time.Now().UTC()
	file := domain.File{
		ID:          fileID,
		OwnerID:     ownerID,
		Filename:    filename,
		Extension:   ext,
		ContentType: contentType,
		Bucket:      s.bucket,
		StoragePath: objectKey,
		SizeBytes:   size,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	created, err := s.repo.Create(ctx, file)
	if err != nil {
		s.log.Error("create file metadata failed", logging.String("file_id", fileID), logging.Err(err))
		_ = s.storage.Delete(context.Background(), objectKey)
		return nil, err
	}

	return created, nil
}

func (s *fileService) GetFile(ctx context.Context, fileID string) (*domain.File, error) {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return nil, domain.ErrInvalidFileID
	}

	return s.repo.GetByID(ctx, fileID)
}

func (s *fileService) GetFileURL(ctx context.Context, fileID string) (string, error) {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return "", domain.ErrInvalidFileID
	}

	file, err := s.repo.GetByID(ctx, fileID)
	if err != nil {
		return "", err
	}

	return s.storage.PublicURL(file.StoragePath), nil
}

func (s *fileService) GetFileURLs(ctx context.Context, fileIDs []string) ([]domain.FileURL, error) {
	if len(fileIDs) == 0 {
		return nil, nil
	}

	urls := make([]domain.FileURL, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		fileID = strings.TrimSpace(fileID)
		if fileID == "" {
			return nil, domain.ErrInvalidFileID
		}

		file, err := s.repo.GetByID(ctx, fileID)
		if err != nil {
			return nil, err
		}

		urls = append(urls, domain.FileURL{
			ID:  fileID,
			URL: s.storage.PublicURL(file.StoragePath),
		})
	}

	return urls, nil
}

func (s *fileService) DeleteFile(ctx context.Context, fileID string) error {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return domain.ErrInvalidFileID
	}

	file, err := s.repo.GetByID(ctx, fileID)
	if err != nil {
		return err
	}

	if err := s.storage.Delete(ctx, file.StoragePath); err != nil {
		s.log.Error("delete stored file failed", logging.String("file_id", fileID), logging.Err(err))
		return domain.ErrFailedToDeleteStoredFile
	}
	if err := s.repo.DeleteByID(ctx, fileID); err != nil {
		return err
	}

	return nil
}

func (s *fileService) compensateCreatedFiles(files []*domain.File) {
	for i := len(files) - 1; i >= 0; i-- {
		file := files[i]
		if file == nil {
			continue
		}
		_ = s.repo.DeleteByID(context.Background(), file.ID)
		_ = s.storage.Delete(context.Background(), file.StoragePath)
	}
}

func buildObjectKey(prefix, fileID, ext string) string {
	if ext == "" {
		return prefix + "/" + fileID
	}

	return prefix + "/" + fileID + "." + ext
}
