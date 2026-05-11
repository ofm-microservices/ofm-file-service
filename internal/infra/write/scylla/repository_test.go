package scylla

import (
	"context"
	"errors"
	"time"

	"file-service/internal/domain"
	"file-service/internal/infra/write/scylla/model"
	"github.com/gocql/gocql"
	"github.com/ofm-microservices/ofm-common/pkg/logging"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type repoQuery struct {
	stmt           string
	execErr        error
	scanErr        error
	scanRow        model.FileRow
	scanCalled     int
	execCalled     int
	ctxCalled      int
	consistency    gocql.Consistency
	consistencySet bool
}

func (q *repoQuery) WithContext(context.Context) query {
	q.ctxCalled++
	return q
}

func (q *repoQuery) Consistency(consistency gocql.Consistency) query {
	q.consistency = consistency
	q.consistencySet = true
	return q
}

func (q *repoQuery) Exec() error {
	q.execCalled++
	return q.execErr
}

func (q *repoQuery) Scan(dest ...any) error {
	q.scanCalled++
	if q.scanErr != nil {
		return q.scanErr
	}
	values := []any{
		&q.scanRow.ID,
		&q.scanRow.OwnerID,
		&q.scanRow.Filename,
		&q.scanRow.Extension,
		&q.scanRow.ContentType,
		&q.scanRow.Bucket,
		&q.scanRow.StoragePath,
		&q.scanRow.SizeBytes,
		&q.scanRow.CreatedAt,
		&q.scanRow.UpdatedAt,
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = *values[i].(*string)
		case *int64:
			*d = *values[i].(*int64)
		case *time.Time:
			*d = *values[i].(*time.Time)
		default:
			panic("unsupported scan destination")
		}
	}
	return nil
}

type repoSession struct {
	queries map[string]*repoQuery
	seen    []string
}

func (s *repoSession) Query(stmt string, _ ...any) query {
	s.seen = append(s.seen, stmt)
	if q, ok := s.queries[stmt]; ok {
		q.stmt = stmt
		return q
	}
	return &repoQuery{stmt: stmt}
}

var _ = Describe("New", func() {
	It("rejects a nil session", func() {
		logger, err := logging.New("file-service", "test", "debug")
		Expect(err).NotTo(HaveOccurred())

		repo, err := New(nil, logger)
		Expect(repo).To(BeNil())
		Expect(err).To(MatchError(ErrNilScyllaDB))
	})
})

var _ = Describe("wrappers", func() {
	It("wraps repository errors", func() {
		Expect(WrapCreateFileError(errors.New("boom")).Error()).To(ContainSubstring("create file"))
		Expect(WrapFindFileError(errors.New("boom")).Error()).To(ContainSubstring("find file"))
		Expect(WrapDeleteFileError(errors.New("boom")).Error()).To(ContainSubstring("delete file"))
	})
})

var _ = Describe("repo operations", func() {
	const (
		createStmt = createFileQuery
		getStmt    = getFileByIDQuery
		deleteStmt = deleteFileByIDQuery
	)

	var logger logging.Logger

	BeforeEach(func() {
		var err error
		logger, err = logging.New("file-service", "test", "debug")
		Expect(err).NotTo(HaveOccurred())
	})

	It("creates and reloads a file", func() {
		now := time.Unix(123, 456).UTC()
		logger, err := logging.New("file-service", "test", "debug")
		Expect(err).NotTo(HaveOccurred())
		session := &repoSession{
			queries: map[string]*repoQuery{
				createStmt: &repoQuery{scanRow: model.FileRow{}},
				getStmt: &repoQuery{
					scanRow: model.FileRow{
						ID:          "file-1",
						OwnerID:     "owner-1",
						Filename:    "cover.png",
						Extension:   "png",
						ContentType: "image/png",
						Bucket:      "bucket",
						StoragePath: "gallery/file-1.png",
						SizeBytes:   3,
						CreatedAt:   now,
						UpdatedAt:   now,
					},
				},
			},
		}
		repo := &repo{db: session, log: logger}

		file, err := repo.Create(context.Background(), domain.File{
			ID:          "file-1",
			OwnerID:     "owner-1",
			Filename:    "cover.png",
			Extension:   "png",
			ContentType: "image/png",
			Bucket:      "bucket",
			StoragePath: "gallery/file-1.png",
			SizeBytes:   3,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(file).NotTo(BeNil())
		Expect(file.ID).To(Equal("file-1"))
		Expect(session.seen).To(Equal([]string{createStmt, getStmt}))
		Expect(session.queries[createStmt].execCalled).To(Equal(1))
		Expect(session.queries[getStmt].scanCalled).To(Equal(1))
	})

	It("wraps create failures", func() {
		session := &repoSession{
			queries: map[string]*repoQuery{
				createStmt: &repoQuery{execErr: errors.New("boom")},
			},
		}
		repo := &repo{db: session, log: logger}

		file, err := repo.Create(context.Background(), domain.File{ID: "file-1"})
		Expect(file).To(BeNil())
		Expect(err).To(MatchError(ContainSubstring("create file")))
	})

	It("loads a file by id", func() {
		now := time.Unix(123, 456).UTC()
		session := &repoSession{
			queries: map[string]*repoQuery{
				getStmt: &repoQuery{
					scanRow: model.FileRow{
						ID:          "file-1",
						OwnerID:     "owner-1",
						Filename:    "cover.png",
						Extension:   "png",
						ContentType: "image/png",
						Bucket:      "bucket",
						StoragePath: "gallery/file-1.png",
						SizeBytes:   3,
						CreatedAt:   now,
						UpdatedAt:   now,
					},
				},
			},
		}
		repo := &repo{db: session}

		file, err := repo.GetByID(context.Background(), "file-1")
		Expect(err).NotTo(HaveOccurred())
		Expect(file.ID).To(Equal("file-1"))
		Expect(session.queries[getStmt].consistencySet).To(BeTrue())
		Expect(session.queries[getStmt].consistency).To(Equal(gocql.One))
	})

	It("maps not found and lookup errors", func() {
		session := &repoSession{
			queries: map[string]*repoQuery{
				getStmt: &repoQuery{scanErr: gocql.ErrNotFound},
			},
		}
		repo := &repo{db: session}

		file, err := repo.GetByID(context.Background(), "missing")
		Expect(file).To(BeNil())
		Expect(err).To(MatchError(domain.ErrFileNotFound))

		session.queries[getStmt].scanErr = errors.New("boom")
		file, err = repo.GetByID(context.Background(), "boom")
		Expect(file).To(BeNil())
		Expect(err).To(MatchError(ContainSubstring("find file")))
	})

	It("deletes a file after confirming it exists", func() {
		now := time.Unix(123, 456).UTC()
		session := &repoSession{
			queries: map[string]*repoQuery{
				getStmt: &repoQuery{
					scanRow: model.FileRow{
						ID:          "file-1",
						OwnerID:     "owner-1",
						Filename:    "cover.png",
						Extension:   "png",
						ContentType: "image/png",
						Bucket:      "bucket",
						StoragePath: "gallery/file-1.png",
						SizeBytes:   3,
						CreatedAt:   now,
						UpdatedAt:   now,
					},
				},
				deleteStmt: &repoQuery{},
			},
		}
		repo := &repo{db: session}

		Expect(repo.DeleteByID(context.Background(), "file-1")).To(Succeed())
		Expect(session.queries[deleteStmt].execCalled).To(Equal(1))
	})

	It("returns lookup and delete failures", func() {
		session := &repoSession{
			queries: map[string]*repoQuery{
				getStmt:    &repoQuery{scanErr: errors.New("missing")},
				deleteStmt: &repoQuery{execErr: errors.New("boom")},
			},
		}
		repo := &repo{db: session}

		Expect(repo.DeleteByID(context.Background(), "missing")).To(MatchError(ContainSubstring("find file")))

		session.queries[getStmt].scanErr = nil
		session.queries[getStmt].scanRow = model.FileRow{ID: "file-1", StoragePath: "gallery/file-1"}
		Expect(repo.DeleteByID(context.Background(), "file-1")).To(MatchError(ContainSubstring("delete file")))
	})
})

var _ = Describe("adapters", func() {
	It("wraps the gocql session and query chain", func() {
		adapter := sessionAdapter{db: &gocql.Session{}}
		q := adapter.Query("SELECT 1")
		Expect(q).NotTo(BeNil())
		Expect(q.WithContext(context.Background())).NotTo(BeNil())
		Expect(q.Consistency(gocql.One)).NotTo(BeNil())
	})
})
