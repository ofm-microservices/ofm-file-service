package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"file-service/internal/domain"
	"github.com/jmoiron/sqlx"
	"github.com/ofm-microservices/ofm-common/pkg/logging"
)

type row struct {
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

// New constructs the PostgreSQL-backed file repository.
func New(db *sqlx.DB, log logging.Logger) (domain.FileRepository, error) {
	if db == nil {
		return nil, ErrNilPostgresDB
	}
	if log == nil {
		return nil, ErrNilLogger
	}
	return &repo{db: db, log: log.With(logging.String("module", "postgres-file-repository"))}, nil
}

type repo struct {
	db  *sqlx.DB
	log logging.Logger
}

func (r *repo) Create(ctx context.Context, file domain.File) (*domain.File, error) {
	_, err := r.db.ExecContext(ctx, `INSERT INTO files(file_id,owner_id,filename,extension,content_type,bucket,storage_path,size_bytes,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, file.ID, file.OwnerID, file.Filename, file.Extension, file.ContentType, file.Bucket, file.StoragePath, file.SizeBytes, file.CreatedAt, file.UpdatedAt)
	if err != nil {
		return nil, WrapCreateFileError(err)
	}
	return r.GetByID(ctx, file.ID)
}
func (r *repo) GetByID(ctx context.Context, id string) (*domain.File, error) {
	var x row
	err := r.db.GetContext(ctx, &x, `SELECT file_id,owner_id,filename,extension,content_type,bucket,storage_path,size_bytes,created_at,updated_at FROM files WHERE file_id=$1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrFileNotFound
	}
	if err != nil {
		return nil, WrapFindFileError(err)
	}
	return &domain.File{ID: x.ID, OwnerID: x.OwnerID, Filename: x.Filename, Extension: x.Extension, ContentType: x.ContentType, Bucket: x.Bucket, StoragePath: x.StoragePath, SizeBytes: x.SizeBytes, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt}, nil
}
func (r *repo) DeleteByID(ctx context.Context, id string) error {
	if _, err := r.GetByID(ctx, id); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `DELETE FROM files WHERE file_id=$1`, id); err != nil {
		return WrapDeleteFileError(err)
	}
	return nil
}
