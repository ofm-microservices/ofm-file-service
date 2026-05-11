package scylla

import (
	"context"
	"errors"
	"file-service/internal/domain"
	"file-service/internal/infra/write/scylla/mapper"
	"file-service/internal/infra/write/scylla/model"
	"time"

	"github.com/gocql/gocql"
	"github.com/ofm-microservices/ofm-common/pkg/logging"
	"github.com/ofm-microservices/ofm-common/pkg/observability/metrics"
)

type repo struct {
	db  session
	log logging.Logger
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
func New(db *gocql.Session, log logging.Logger) (domain.FileRepository, error) {
	if db == nil {
		return nil, ErrNilScyllaDB
	}
	if log == nil {
		return nil, ErrNilLogger
	}

	return &repo{db: sessionAdapter{db: db}, log: log.With(logging.String("module", "scylla-repository"))}, nil
}

func (r *repo) Create(ctx context.Context, file domain.File) (*domain.File, error) {
	started := time.Now()
	status := "success"
	defer func() { metrics.Global().ObserveDB("scylla", "create", "files", status, time.Since(started)) }()

	row := mapper.MapDomainFileToRow(file)
	if err := r.db.Query(
		createFileQuery,
		row.ID, row.OwnerID, row.Filename, row.Extension, row.ContentType, row.Bucket, row.StoragePath, row.SizeBytes, row.CreatedAt, row.UpdatedAt,
	).WithContext(ctx).Exec(); err != nil {
		status = "error"
		r.log.Error("create file failed",
			logging.Operation("db.file.create"),
			logging.Attempt(1),
			logging.Retryable(false),
			logging.DurationMS(time.Since(started)),
			logging.String("file_id", file.ID),
			logging.Err(err),
		)
		return nil, WrapCreateFileError(err)
	}

	created, err := r.GetByID(ctx, file.ID)
	if err != nil {
		status = "error"
	}
	return created, err
}

func (r *repo) GetByID(ctx context.Context, fileID string) (*domain.File, error) {
	started := time.Now()
	status := "success"
	defer func() { metrics.Global().ObserveDB("scylla", "get_by_id", "files", status, time.Since(started)) }()

	var row model.FileRow
	if err := r.db.Query(getFileByIDQuery, fileID).WithContext(ctx).Consistency(gocql.One).
		Scan(&row.ID, &row.OwnerID, &row.Filename, &row.Extension, &row.ContentType, &row.Bucket, &row.StoragePath, &row.SizeBytes, &row.CreatedAt, &row.UpdatedAt); err != nil {
		status = "error"
		r.log.Error("get file failed",
			logging.Operation("db.file.get_by_id"),
			logging.Attempt(1),
			logging.Retryable(false),
			logging.DurationMS(time.Since(started)),
			logging.String("file_id", fileID),
			logging.Err(err),
		)
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, domain.ErrFileNotFound
		}
		return nil, WrapFindFileError(err)
	}

	return mapper.MapRowToDomainFile(row), nil
}

func (r *repo) DeleteByID(ctx context.Context, fileID string) error {
	started := time.Now()
	status := "success"
	defer func() { metrics.Global().ObserveDB("scylla", "delete_by_id", "files", status, time.Since(started)) }()

	if _, err := r.GetByID(ctx, fileID); err != nil {
		status = "error"
		r.log.Error("delete file lookup failed",
			logging.Operation("db.file.delete_by_id"),
			logging.Attempt(1),
			logging.Retryable(false),
			logging.DurationMS(time.Since(started)),
			logging.String("file_id", fileID),
			logging.Err(err),
		)
		return err
	}
	if err := r.db.Query(deleteFileByIDQuery, fileID).WithContext(ctx).Exec(); err != nil {
		status = "error"
		r.log.Error("delete file failed",
			logging.Operation("db.file.delete_by_id"),
			logging.Attempt(1),
			logging.Retryable(false),
			logging.DurationMS(time.Since(started)),
			logging.String("file_id", fileID),
			logging.Err(err),
		)
		return WrapDeleteFileError(err)
	}

	return nil
}
