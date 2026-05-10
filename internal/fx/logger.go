package appfx

import (
	"file-service/config"

	"github.com/ofm-microseervices/ofm-common/pkg/logging"
	"go.uber.org/fx"
)

// LoggerModule provides the structured logger used throughout file-service.
var LoggerModule = fx.Options(
	fx.Provide(ProvideLogger),
)

// ProvideLogger constructs the shared structured logger.
func ProvideLogger(cfg *config.Config) (logging.Logger, error) {
	return logging.New("file-service", cfg.App.Env, cfg.App.LogLevel)
}
