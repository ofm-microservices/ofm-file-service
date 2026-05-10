package appfx

import (
	"context"

	"file-service/config"
	app "file-service/internal/application"
	grpcserver "file-service/internal/presentation/grpc"
	"github.com/ofm-microseervices/ofm-common/pkg/logging"

	"go.uber.org/fx"
)

// PresentationModule wires the transport adapters used by the service.
var PresentationModule = fx.Options(
	fx.Provide(
		ProvideGRPCServer,
	),
	fx.Invoke(
		InvokeRunGRPCServer,
	),
)

// ProvideGRPCServer constructs the internal gRPC server used by the service.
func ProvideGRPCServer(service app.FileService, cfg *config.Config, lg logging.Logger) (grpcserver.Server, error) {
	return grpcserver.NewServer(service, cfg.GRPC, lg)
}

// InvokeRunGRPCServer starts and gracefully stops the gRPC server with the FX
// lifecycle.
func InvokeRunGRPCServer(lc fx.Lifecycle, srv grpcserver.Server) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := srv.Start(); err != nil {
					panic(err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
}
