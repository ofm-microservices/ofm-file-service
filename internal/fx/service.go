package appfx

import (
	"file-service/config"
	app "file-service/internal/application"
	"file-service/internal/domain"
	"github.com/ofm-microservices/ofm-common/pkg/logging"

	"go.uber.org/fx"
)

// ServiceModule provides the application service.
var ServiceModule = fx.Options(
	fx.Provide(ProvideFileService),
)

// ProvideFileService constructs the file application service.
func ProvideFileService(repo domain.FileRepository, storage domain.FileStorage, lg logging.Logger, cfg *config.Config) (app.FileService, error) {
	return app.New(repo, storage, cfg.RustFS.Bucket, lg)
}
