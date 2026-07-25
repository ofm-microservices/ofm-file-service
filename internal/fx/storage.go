package appfx

import (
	"context"

	"file-service/config"
	"file-service/internal/domain"
	pkgrustfs "file-service/pkg/storage/rustfs"
	pkgscylla "file-service/pkg/storage/scylla"
	"github.com/ofm-microservices/ofm-common/pkg/logging"

	"github.com/gocql/gocql"
	"go.uber.org/fx"
)

// StorageModule provides the primary ScyllaDB and RustFS clients used by
// file-service.
var StorageModule = fx.Options(
	fx.Provide(
		ProvideScyllaSession,
		ProvideRustFSStorage,
	),
)

var connectScylla = pkgscylla.ConnectAndEnsureSchema
var openRustFS = pkgrustfs.Open

// ProvideScyllaSession connects to ScyllaDB, ensures the schema exists, and
// closes the session on shutdown.
func ProvideScyllaSession(lc fx.Lifecycle, cfg *config.Config, lg logging.Logger) (*gocql.Session, error) {
	session, err := connectScylla(pkgscylla.Options{
		Hosts:                  cfg.Scylla.Hosts,
		Port:                   cfg.Scylla.Port,
		Keyspace:               cfg.Scylla.Keyspace,
		Username:               cfg.Scylla.Username,
		Password:               cfg.Scylla.Password,
		Consistency:            cfg.Scylla.Consistency,
		ConnectTimeout:         cfg.Scylla.ConnectTimeout,
		MaxWaitSchemaAgreement: cfg.Scylla.MaxWaitSchemaAgreement,
		RetryAttempts:          cfg.Scylla.RetryAttempts,
		RetryBackoff:           cfg.Scylla.RetryBackoff,
	})
	if err != nil {
		lg.Error("connect scylla failed", logging.Err(err))
		return nil, err
	}
	raw := session.Raw()

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			session.Close()
			return nil
		},
	})

	return raw, nil
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
