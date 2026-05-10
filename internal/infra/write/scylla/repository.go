package scylla

import (
	"context"
	"errors"
	"file-service/internal/domain"
	"file-service/internal/infra/write/scylla/mapper"
	"file-service/internal/infra/write/scylla/model"

	"github.com/gocql/gocql"
)

type repo struct {
	db session
}

type session interface {
	Query(stmt string, values ...any) query
}

type query interface {
	WithContext(ctx context.Context) query
	Consistency(consistency gocql.Consistency) query
	Exec() error
	Scan(dest ...any) error
}

type sessionAdapter struct {
	db *gocql.Session
}

func (s sessionAdapter) Query(stmt string, values ...any) query {
	return queryAdapter{q: s.db.Query(stmt, values...)}
}

type queryAdapter struct {
	q *gocql.Query
}

func (q queryAdapter) WithContext(ctx context.Context) query {
	q.q = q.q.WithContext(ctx)
	return q
}

func (q queryAdapter) Consistency(consistency gocql.Consistency) query {
	q.q = q.q.Consistency(consistency)
	return q
}

func (q queryAdapter) Exec() error {
	return q.q.Exec()
}

func (q queryAdapter) Scan(dest ...any) error {
	return q.q.Scan(dest...)
}

// New constructs the Scylla-backed file repository.
func New(db *gocql.Session) (domain.FileRepository, error) {
	if db == nil {
		return nil, ErrNilScyllaDB
	}

	return &repo{db: sessionAdapter{db: db}}, nil
}

func (r *repo) Create(ctx context.Context, file domain.File) (*domain.File, error) {
	row := mapper.MapDomainFileToRow(file)
	if err := r.db.Query(
		createFileQuery,
		row.ID, row.OwnerID, row.Filename, row.Extension, row.ContentType, row.Bucket, row.StoragePath, row.SizeBytes, row.CreatedAt, row.UpdatedAt,
	).WithContext(ctx).Exec(); err != nil {
		return nil, WrapCreateFileError(err)
	}

	return r.GetByID(ctx, file.ID)
}

func (r *repo) GetByID(ctx context.Context, fileID string) (*domain.File, error) {
	var row model.FileRow
	if err := r.db.Query(getFileByIDQuery, fileID).WithContext(ctx).Consistency(gocql.One).
		Scan(&row.ID, &row.OwnerID, &row.Filename, &row.Extension, &row.ContentType, &row.Bucket, &row.StoragePath, &row.SizeBytes, &row.CreatedAt, &row.UpdatedAt); err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, domain.ErrFileNotFound
		}
		return nil, WrapFindFileError(err)
	}

	return mapper.MapRowToDomainFile(row), nil
}

func (r *repo) DeleteByID(ctx context.Context, fileID string) error {
	if _, err := r.GetByID(ctx, fileID); err != nil {
		return err
	}
	if err := r.db.Query(deleteFileByIDQuery, fileID).WithContext(ctx).Exec(); err != nil {
		return WrapDeleteFileError(err)
	}

	return nil
}
