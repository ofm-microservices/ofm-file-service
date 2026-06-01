package appfx

import (
	"context"
	"errors"
	"os"
	"time"

	"file-service/config"
	"file-service/internal/domain"
	writerepo "file-service/internal/infra/write/scylla"
	pkgrustfs "file-service/pkg/storage/rustfs"
	pkgscylla "file-service/pkg/storage/scylla"
	"github.com/gocql/gocql"
	"github.com/ofm-microservices/ofm-common/pkg/logging"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
)

type loggerStub struct {
	infos  []string
	errors []string
}

func (l *loggerStub) Debug(string, ...logging.Field)         {}
func (l *loggerStub) Info(msg string, _ ...logging.Field)    { l.infos = append(l.infos, msg) }
func (l *loggerStub) Warn(string, ...logging.Field)          {}
func (l *loggerStub) Error(msg string, _ ...logging.Field)   { l.errors = append(l.errors, msg) }
func (l *loggerStub) With(_ ...logging.Field) logging.Logger { return l }
func (l *loggerStub) Sync() error                            { return nil }

type repoStub struct{}

func (r *repoStub) Create(context.Context, domain.File) (*domain.File, error) { return nil, nil }
func (r *repoStub) GetByID(context.Context, string) (*domain.File, error)     { return nil, nil }
func (r *repoStub) DeleteByID(context.Context, string) error                  { return nil }

type storageStub struct{}

func (s *storageStub) Put(context.Context, string, string, []byte) (int64, error) { return 0, nil }
func (s *storageStub) PresignPut(context.Context, string, string) (string, error) {
	return "http://upload.local", nil
}
func (s *storageStub) Delete(context.Context, string) error { return nil }
func (s *storageStub) PublicURL(string) string              { return "http://public.local/file-1" }

type lifecycleStub struct {
	hooks []fx.Hook
}

func (l *lifecycleStub) Append(h fx.Hook) { l.hooks = append(l.hooks, h) }

type scyllaQueryStub struct {
	execErr error
	execCnt int
}

func (q *scyllaQueryStub) WithContext(context.Context) pkgscylla.Query   { return q }
func (q *scyllaQueryStub) Consistency(gocql.Consistency) pkgscylla.Query { return q }
func (q *scyllaQueryStub) Exec() error {
	q.execCnt++
	return q.execErr
}

type scyllaSessionStub struct {
	closeCnt int
	queryFn  func(string) pkgscylla.Query
}

func (s *scyllaSessionStub) Close() { s.closeCnt++ }

func (s *scyllaSessionStub) Query(stmt string, _ ...any) pkgscylla.Query {
	if s.queryFn != nil {
		return s.queryFn(stmt)
	}
	return &scyllaQueryStub{}
}

func (s *scyllaSessionStub) Raw() *gocql.Session { return &gocql.Session{} }

var _ = Describe("providers", func() {
	It("loads config from the environment", func() {
		DeferCleanup(func() {
			_ = os.Unsetenv("APP_ENV")
			_ = os.Unsetenv("RUSTFS_ENDPOINT")
		})
		Expect(os.Setenv("APP_ENV", "test")).To(Succeed())
		Expect(os.Setenv("RUSTFS_ENDPOINT", "http://localhost:9006")).To(Succeed())

		cfg, err := ProvideConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.App.Env).To(Equal("test"))
	})

	It("constructs the logger", func() {
		cfg := &config.Config{App: config.AppConfig{Env: "test", LogLevel: "info"}}
		lg, err := ProvideLogger(cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(lg).NotTo(BeNil())
	})

	It("reports logger construction errors", func() {
		cfg := &config.Config{App: config.AppConfig{Env: "test", LogLevel: "nope"}}
		lg, err := ProvideLogger(cfg)
		Expect(lg).To(BeNil())
		Expect(err).To(HaveOccurred())
	})

	It("constructs the repository", func() {
		repo, err := ProvideFileRepository(&gocql.Session{}, &loggerStub{})
		Expect(err).NotTo(HaveOccurred())
		Expect(repo).NotTo(BeNil())

		repo, err = ProvideFileRepository(nil, &loggerStub{})
		Expect(err).To(MatchError(writerepo.ErrNilScyllaDB))
	})

	It("constructs the application service", func() {
		cfg := &config.Config{RustFS: config.RustFSConfig{Bucket: "bucket"}}
		svc, err := ProvideFileService(&repoStub{}, &storageStub{}, &loggerStub{}, cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(svc).NotTo(BeNil())
	})

	It("constructs the gRPC server", func() {
		cfg := &config.Config{GRPC: config.GRPCConfig{Host: "127.0.0.1", Port: 9504}}
		srv, err := ProvideGRPCServer(&fileServiceStub{}, cfg, &loggerStub{})
		Expect(err).NotTo(HaveOccurred())
		Expect(srv).NotTo(BeNil())
	})

	It("opens RustFS storage and reports failures", func() {
		origOpenRustFS := openRustFS
		defer func() { openRustFS = origOpenRustFS }()
		openRustFS = func(context.Context, pkgrustfs.Options) (pkgrustfs.Storage, error) {
			return &storageStub{}, nil
		}

		cfg := &config.Config{RustFS: config.RustFSConfig{Bucket: "bucket"}}
		storage, err := ProvideRustFSStorage(cfg, &loggerStub{})
		Expect(err).NotTo(HaveOccurred())
		Expect(storage).NotTo(BeNil())

		openRustFS = func(context.Context, pkgrustfs.Options) (pkgrustfs.Storage, error) {
			return nil, errors.New("boom")
		}
		storage, err = ProvideRustFSStorage(cfg, &loggerStub{})
		Expect(storage).To(BeNil())
		Expect(err).To(HaveOccurred())
	})

	It("connects Scylla and closes it on shutdown", func() {
		origConnectScylla := connectScylla
		defer func() { connectScylla = origConnectScylla }()
		sess := &scyllaSessionStub{
			queryFn: func(string) pkgscylla.Query { return &scyllaQueryStub{} },
		}
		connectScylla = func(pkgscylla.Options) (pkgscylla.Session, error) {
			return sess, nil
		}

		lc := &lifecycleStub{}
		cfg := &config.Config{Scylla: config.ScyllaConfig{Hosts: []string{"127.0.0.1"}, Keyspace: "ks"}}
		raw, err := ProvideScyllaSession(lc, cfg, &loggerStub{})
		Expect(err).NotTo(HaveOccurred())
		Expect(raw).NotTo(BeNil())
		Expect(lc.hooks).To(HaveLen(1))
		Expect(lc.hooks[0].OnStop(context.Background())).To(Succeed())
		Expect(sess.closeCnt).To(Equal(1))
	})

	It("reports Scylla connection failures", func() {
		origConnectScylla := connectScylla
		defer func() { connectScylla = origConnectScylla }()
		connectScylla = func(pkgscylla.Options) (pkgscylla.Session, error) {
			return nil, errors.New("boom")
		}
		cfg := &config.Config{Scylla: config.ScyllaConfig{
			Hosts:          []string{"127.0.0.1"},
			Port:           1,
			ConnectTimeout: time.Millisecond,
			RetryAttempts:  1,
			RetryBackoff:   0,
		}}

		session, err := ProvideScyllaSession(&lifecycleStub{}, cfg, &loggerStub{})
		Expect(session).To(BeNil())
		Expect(err).To(HaveOccurred())
	})

	It("logs service startup", func() {
		lg := &loggerStub{}
		InvokeStartLog(lg)
		Expect(lg.infos).To(ContainElement("starting file-service"))
	})
})

type fileServiceStub struct{}

func (fileServiceStub) CreateFile(context.Context, domain.UploadFileParams) (*domain.File, error) {
	return nil, nil
}
func (fileServiceStub) CreateFiles(context.Context, domain.UploadFilesParams) ([]*domain.File, error) {
	return nil, nil
}
func (fileServiceStub) GetFile(context.Context, string) (*domain.File, error) {
	return nil, nil
}
func (fileServiceStub) GetFileURL(context.Context, string) (string, error) {
	return "", nil
}
func (fileServiceStub) GetFileURLs(context.Context, []string) ([]domain.FileURL, error) {
	return nil, nil
}
func (fileServiceStub) DeleteFile(context.Context, string) error { return nil }

var _ pkgrustfs.Storage = (*storageStub)(nil)
