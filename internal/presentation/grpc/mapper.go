package grpc

import (
	"errors"
	"time"

	"file-service/internal/domain"
	"github.com/ofm-microservices/ofm-common/pkg/logging"
	filev1 "github.com/ofm-microservices/ofm-common/proto/file/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FileMapper translates between gRPC messages and the file application model.
type FileMapper interface {
	ToUploadParams(req *filev1.UploadFileRequest) domain.UploadFileParams
	ToUploadFilesParams(req *filev1.UploadFilesRequest) domain.UploadFilesParams
	ToFileResponse(file *domain.File) *filev1.File
	ToUploadResponse(file *domain.File) *filev1.UploadFileResponse
	ToUploadFilesResponse(files []*domain.File) *filev1.UploadFilesResponse
	ToFileURLResponse(fileID, url string) *filev1.GetFileURLResponse
	ToDeleteResponse(fileID string) *filev1.DeleteFileResponse
	ToError(err error) error
}

type fileMapper struct {
	log logging.Logger
}

func newFileMapper(log logging.Logger) FileMapper {
	return &fileMapper{log: log}
}

func (m *fileMapper) ToUploadParams(req *filev1.UploadFileRequest) domain.UploadFileParams {
	return domain.UploadFileParams{
		OwnerID:     req.GetOwnerId(),
		Filename:    req.GetFilename(),
		ContentType: req.GetContentType(),
		Prefix:      req.GetPrefix(),
		Data:        req.GetData(),
	}
}

func (m *fileMapper) ToUploadFilesParams(req *filev1.UploadFilesRequest) domain.UploadFilesParams {
	files := make([]domain.UploadFileParams, 0, len(req.GetFiles()))
	for _, item := range req.GetFiles() {
		files = append(files, domain.UploadFileParams{
			OwnerID:     req.GetOwnerId(),
			Filename:    item.GetFilename(),
			ContentType: item.GetContentType(),
			Prefix:      req.GetPrefix(),
			Data:        item.GetData(),
		})
	}

	return domain.UploadFilesParams{
		OwnerID: req.GetOwnerId(),
		Prefix:  req.GetPrefix(),
		Files:   files,
	}
}

func (m *fileMapper) ToFileResponse(file *domain.File) *filev1.File {
	if file == nil {
		return nil
	}

	return &filev1.File{
		FileId:      file.ID,
		OwnerId:     file.OwnerID,
		Filename:    file.Filename,
		Extension:   file.Extension,
		ContentType: file.ContentType,
		Bucket:      file.Bucket,
		StoragePath: file.StoragePath,
		SizeBytes:   file.SizeBytes,
		CreatedAt:   file.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:   file.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (m *fileMapper) ToUploadResponse(file *domain.File) *filev1.UploadFileResponse {
	return &filev1.UploadFileResponse{File: m.ToFileResponse(file)}
}

func (m *fileMapper) ToUploadFilesResponse(files []*domain.File) *filev1.UploadFilesResponse {
	resp := &filev1.UploadFilesResponse{}
	if len(files) == 0 {
		return resp
	}

	resp.Files = make([]*filev1.File, 0, len(files))
	for _, file := range files {
		resp.Files = append(resp.Files, m.ToFileResponse(file))
	}

	return resp
}

func (m *fileMapper) ToFileURLResponse(fileID, url string) *filev1.GetFileURLResponse {
	return &filev1.GetFileURLResponse{FileId: fileID, Url: url}
}

func (m *fileMapper) ToDeleteResponse(fileID string) *filev1.DeleteFileResponse {
	return &filev1.DeleteFileResponse{FileId: fileID}
}

func (m *fileMapper) ToError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrInvalidFileID),
		errors.Is(err, domain.ErrInvalidOwnerID),
		errors.Is(err, domain.ErrInvalidFilename),
		errors.Is(err, domain.ErrInvalidContentType),
		errors.Is(err, domain.ErrInvalidStoragePath),
		errors.Is(err, domain.ErrInvalidFileData):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrFileNotFound):
		return status.Error(codes.NotFound, "file not found")
	case errors.Is(err, domain.ErrFailedToStoreFile),
		errors.Is(err, domain.ErrFailedToCreateFile),
		errors.Is(err, domain.ErrFailedToFindFile),
		errors.Is(err, domain.ErrFailedToDeleteFile),
		errors.Is(err, domain.ErrFailedToDeleteStoredFile):
		return status.Error(codes.Internal, err.Error())
	default:
		m.log.Error("file request failed",
			logging.Operation("grpc.file.map_error"),
			logging.Attempt(1),
			logging.Retryable(false),
			logging.Err(err),
		)
		return status.Error(codes.Internal, "internal server error")
	}
}
