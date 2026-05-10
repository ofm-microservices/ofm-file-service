package grpc

import (
	"context"

	app "file-service/internal/application"
)

// FileService exposes the application file workflow to the gRPC transport.
type FileService = app.FileService

// Server defines the gRPC server lifecycle exposed by file-service.
type Server interface {
	Start() error
	Shutdown(ctx context.Context) error
}
