package grpc

import (
	"context"
	"fmt"
	"net"

	"file-service/config"
	"github.com/ofm-microservices/ofm-common/pkg/logging"
	"github.com/ofm-microservices/ofm-common/pkg/observability/metrics"
	filev1 "github.com/ofm-microservices/ofm-common/proto/file/v1"
	"google.golang.org/grpc"
)

type server struct {
	filev1.UnimplementedFileServiceServer
	svc      FileService
	cfg      config.GRPCConfig
	log      logging.Logger
	mapr     FileMapper
	srv      *grpc.Server
	listener net.Listener
}

// NewServer constructs the file-service gRPC server.
func NewServer(svc FileService, cfg config.GRPCConfig, log logging.Logger) (Server, error) {
	if svc == nil {
		return nil, ErrNilFileService
	}
	if log == nil {
		return nil, ErrNilLogger
	}

	grpcSrv := grpc.NewServer(grpc.UnaryInterceptor(metrics.UnaryServerInterceptor()))
	s := &server{
		svc:  svc,
		cfg:  cfg,
		mapr: newFileMapper(log.With(logging.String("module", "grpc-mapper"))),
		log:  log.With(logging.String("module", "grpc-server")),
		srv:  grpcSrv,
	}
	filev1.RegisterFileServiceServer(grpcSrv, s)
	return s, nil
}

// Start begins serving gRPC traffic on the configured address.
func (s *server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.listener = lis
	s.log.Info("starting grpc server", logging.String("addr", addr))
	return s.srv.Serve(lis)
}

// Shutdown gracefully stops the gRPC server.
func (s *server) Shutdown(context.Context) error {
	if s.srv != nil {
		s.log.Info("shutting down grpc server")
		s.srv.GracefulStop()
	}
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// UploadFile stores a file object and its metadata.
func (s *server) UploadFile(ctx context.Context, req *filev1.UploadFileRequest) (*filev1.UploadFileResponse, error) {
	file, err := s.svc.CreateFile(ctx, s.mapr.ToUploadParams(req))
	if err != nil {
		return nil, s.mapr.ToError(err)
	}

	return s.mapr.ToUploadResponse(file), nil
}

// UploadFiles stores a batch of file objects and publishes their creation as a
// single event.
func (s *server) UploadFiles(ctx context.Context, req *filev1.UploadFilesRequest) (*filev1.UploadFilesResponse, error) {
	files, err := s.svc.CreateFiles(ctx, s.mapr.ToUploadFilesParams(req))
	if err != nil {
		return nil, s.mapr.ToError(err)
	}

	return s.mapr.ToUploadFilesResponse(files), nil
}

// GetFile returns the file metadata from the write model.
func (s *server) GetFile(ctx context.Context, req *filev1.GetFileRequest) (*filev1.GetFileResponse, error) {
	file, err := s.svc.GetFile(ctx, req.GetFileId())
	if err != nil {
		return nil, s.mapr.ToError(err)
	}

	return &filev1.GetFileResponse{File: s.mapr.ToFileResponse(file)}, nil
}

// DeleteFile deletes the file object and its metadata.
func (s *server) DeleteFile(ctx context.Context, req *filev1.DeleteFileRequest) (*filev1.DeleteFileResponse, error) {
	if err := s.svc.DeleteFile(ctx, req.GetFileId()); err != nil {
		return nil, s.mapr.ToError(err)
	}

	return s.mapr.ToDeleteResponse(req.GetFileId()), nil
}
