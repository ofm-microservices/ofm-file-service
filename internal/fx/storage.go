package appfx

import (
	"context"

	"file-service/config"
	"file-service/internal/domain"
	pkgpostgres "file-service/pkg/storage/postgres"
	pkgrustfs "file-service/pkg/storage/rustfs"
	"github.com/ofm-microservices/ofm-common/pkg/logging"

	"github.com/jmoiron/sqlx"
	"go.uber.org/fx"
)

// StorageModule provides the PostgreSQL and RustFS clients used by file-service.
var StorageModule = fx.Options(
	fx.Provide(
		ProvidePostgresDB,
		ProvideRustFSStorage,
	),
)

var openRustFS = pkgrustfs.Open

// ProvidePostgresDB opens the file-service PostgreSQL database.
func ProvidePostgresDB(lc fx.Lifecycle, cfg *config.Config, lg logging.Logger) (*sqlx.DB, error) {
	if err := pkgpostgres.RunMigrations(cfg.DB); err != nil {
		return nil, err
	}
	db, err := pkgpostgres.Open(cfg.DB)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{OnStop: func(context.Context) error { return db.Close() }})
	lg.Info("PostgreSQL connected")
	return db, nil
}

// ProvideRustFSStorage opens the RustFS object storage adapter.
func ProvideRustFSStorage(cfg *config.Config, lg logging.Logger) (domain.FileStorage, error) {
	storage, err := openRustFS(context.Background(), pkgrustfs.Options{
		Endpoint:  cfg.RustFS.Endpoint,
		AccessKey: cfg.RustFS.AccessKey,
		SecretKey: cfg.RustFS.SecretKey,
		Region:    cfg.RustFS.Region,
		Bucket:    cfg.RustFS.Bucket,
		Secure:    cfg.RustFS.Secure,
	})
	if err != nil {
		lg.Error("open rustfs failed", logging.Err(err))
		return nil, err
	}

	return storage, nil
}
