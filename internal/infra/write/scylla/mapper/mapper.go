package mapper

import (
	"file-service/internal/domain"
	"file-service/internal/infra/write/scylla/model"
)

// MapDomainFileToRow converts a domain file into its ScyllaDB row model.
func MapDomainFileToRow(file domain.File) model.FileRow {
	return model.FileRow{
		ID:          file.ID,
		OwnerID:     file.OwnerID,
		Filename:    file.Filename,
		Extension:   file.Extension,
		ContentType: file.ContentType,
		Bucket:      file.Bucket,
		StoragePath: file.StoragePath,
		SizeBytes:   file.SizeBytes,
		CreatedAt:   file.CreatedAt,
		UpdatedAt:   file.UpdatedAt,
	}
}

// MapRowToDomainFile converts a ScyllaDB row into the domain file model.
func MapRowToDomainFile(row model.FileRow) *domain.File {
	return &domain.File{
		ID:          row.ID,
		OwnerID:     row.OwnerID,
		Filename:    row.Filename,
		Extension:   row.Extension,
		ContentType: row.ContentType,
		Bucket:      row.Bucket,
		StoragePath: row.StoragePath,
		SizeBytes:   row.SizeBytes,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
