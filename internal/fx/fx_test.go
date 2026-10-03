package appfx

import (
	"context"
	"errors"
	"os"

	"file-service/config"
	"file-service/internal/domain"
	pkgrustfs "file-service/pkg/storage/rustfs"
	"github.com/jmoiron/sqlx"
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
func (s *storageStub) Exists(context.Context, string) (bool, error) { return true, nil }
func (s *storageStub) Delete(context.Context, string) error         { return nil }
func (s *storageStub) PublicURL(string) string                      { return "http://public.local/file-1" }

type lifecycleStub struct {
	hooks []fx.Hook
}

func (l *lifecycleStub) Append(h fx.Hook) { l.hooks = append(l.hooks, h) }

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
		repo, err := ProvidePostgresFileRepository(&sqlx.DB{}, &loggerStub{})
		Expect(err).NotTo(HaveOccurred())
		Expect(repo).NotTo(BeNil())

		repo, err = ProvidePostgresFileRepository(nil, &loggerStub{})
		Expect(err).To(MatchError("postgres database is nil"))
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
func (fileServiceStub) CreateDirectUpload(context.Context, domain.CreateDirectUploadParams) (*domain.File, string, error) {
	return nil, "", nil
}
func (fileServiceStub) CompleteDirectUpload(context.Context, string) (*domain.File, string, error) {
	return nil, "", nil
}
func (fileServiceStub) GetFileURL(context.Context, string) (string, error) {
	return "", nil
}
func (fileServiceStub) GetFileURLs(context.Context, []string) ([]domain.FileURL, error) {
	return nil, nil
}
func (fileServiceStub) DeleteFile(context.Context, string) error { return nil }

var _ pkgrustfs.Storage = (*storageStub)(nil)
