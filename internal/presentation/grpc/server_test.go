package grpc

import (
	"context"
	"errors"
	"time"

	"file-service/config"
	"file-service/internal/domain"
	"github.com/google/uuid"
	"github.com/ofm-microservices/ofm-common/pkg/logging"
	filev1 "github.com/ofm-microservices/ofm-common/proto/file/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fileSvcFake struct {
	createFileFn  func(context.Context, domain.UploadFileParams) (*domain.File, error)
	createFilesFn func(context.Context, domain.UploadFilesParams) ([]*domain.File, error)
	getFileFn     func(context.Context, string) (*domain.File, error)
	deleteFileFn  func(context.Context, string) error
	getURLFn      func(context.Context, string) (string, error)
	getURLsFn     func(context.Context, []string) ([]domain.FileURL, error)
	createArgs    []domain.UploadFileParams
	createMany    []domain.UploadFilesParams
	getIDs        []string
	deleteIDs     []string
}

func (f *fileSvcFake) CreateFile(ctx context.Context, params domain.UploadFileParams) (*domain.File, error) {
	f.createArgs = append(f.createArgs, params)
	if f.createFileFn != nil {
		return f.createFileFn(ctx, params)
	}
	return &domain.File{ID: "file-1", OwnerID: params.OwnerID, Filename: params.Filename}, nil
}

func (f *fileSvcFake) CreateFiles(ctx context.Context, params domain.UploadFilesParams) ([]*domain.File, error) {
	f.createMany = append(f.createMany, params)
	if f.createFilesFn != nil {
		return f.createFilesFn(ctx, params)
	}
	return []*domain.File{}, nil
}

func (f *fileSvcFake) GetFile(ctx context.Context, fileID string) (*domain.File, error) {
	f.getIDs = append(f.getIDs, fileID)
	if f.getFileFn != nil {
		return f.getFileFn(ctx, fileID)
	}
	return &domain.File{ID: fileID}, nil
}

func (f *fileSvcFake) DeleteFile(ctx context.Context, fileID string) error {
	f.deleteIDs = append(f.deleteIDs, fileID)
	if f.deleteFileFn != nil {
		return f.deleteFileFn(ctx, fileID)
	}
	return nil
}

func (f *fileSvcFake) GetFileURL(ctx context.Context, fileID string) (string, error) {
	if f.getURLFn != nil {
		return f.getURLFn(ctx, fileID)
	}
	return "http://public.local/" + fileID, nil
}

func (f *fileSvcFake) GetFileURLs(ctx context.Context, fileIDs []string) ([]domain.FileURL, error) {
	if f.getURLsFn != nil {
		return f.getURLsFn(ctx, fileIDs)
	}

	urls := make([]domain.FileURL, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		urls = append(urls, domain.FileURL{ID: fileID, URL: "http://public.local/" + fileID})
	}

	return urls, nil
}

func testLogger() logging.Logger {
	lg, err := logging.New("file-service-test", "test", "error")
	Expect(err).NotTo(HaveOccurred())
	return lg
}

var _ = Describe("NewServer", func() {
	It("rejects missing dependencies", func() {
		_, err := NewServer(nil, config.GRPCConfig{}, testLogger())
		Expect(err).To(MatchError(ErrNilFileService))

		_, err = NewServer(&fileSvcFake{}, config.GRPCConfig{}, nil)
		Expect(err).To(MatchError(ErrNilLogger))
	})

	It("constructs the server", func() {
		srv, err := NewServer(&fileSvcFake{}, config.GRPCConfig{}, testLogger())
		Expect(err).NotTo(HaveOccurred())
		Expect(srv).NotTo(BeNil())
	})
})

var _ = Describe("server lifecycle", func() {
	It("returns an error when the listener cannot be created", func() {
		srv := &server{
			svc: &fileSvcFake{},
			cfg: config.GRPCConfig{Host: "invalid host", Port: 9504},
			log: testLogger(),
			srv: grpc.NewServer(),
		}

		Expect(srv.Start()).To(HaveOccurred())
	})

	It("shuts down without a listener", func() {
		srv := &server{srv: grpc.NewServer(), log: testLogger()}
		Expect(srv.Shutdown(context.Background())).To(Succeed())
	})
})

var _ = Describe("RPC handlers", func() {
	It("uploads a file", func() {
		svc := &fileSvcFake{
			createFileFn: func(ctx context.Context, params domain.UploadFileParams) (*domain.File, error) {
				Expect(params.OwnerID).To(Equal("owner-1"))
				Expect(params.Filename).To(Equal("cover.png"))
				Expect(params.ContentType).To(Equal("image/png"))
				Expect(params.Prefix).To(Equal("gallery"))
				Expect(params.Data).To(Equal([]byte("abc")))
				return &domain.File{
					ID:          uuid.Must(uuid.NewV7()).String(),
					OwnerID:     params.OwnerID,
					Filename:    params.Filename,
					Extension:   "png",
					ContentType: params.ContentType,
					Bucket:      "bucket",
					StoragePath: "gallery/file-1.png",
					SizeBytes:   3,
					CreatedAt:   time.Unix(1, 234).UTC(),
					UpdatedAt:   time.Unix(2, 345).UTC(),
				}, nil
			},
		}
		srv := &server{svc: svc, log: testLogger(), mapr: newFileMapper(testLogger())}

		resp, err := srv.UploadFile(context.Background(), &filev1.UploadFileRequest{
			OwnerId:     "owner-1",
			Filename:    "cover.png",
			ContentType: "image/png",
			Prefix:      "gallery",
			Data:        []byte("abc"),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.GetFile().GetOwnerId()).To(Equal("owner-1"))
		Expect(resp.GetFile().GetFilename()).To(Equal("cover.png"))
		Expect(resp.GetFile().GetExtension()).To(Equal("png"))
		Expect(resp.GetFile().GetCreatedAt()).To(HavePrefix("1970-01-01T00:00:01"))
		Expect(svc.createArgs).To(HaveLen(1))
	})

	It("propagates upload file errors", func() {
		srv := &server{
			svc: &fileSvcFake{
				createFileFn: func(ctx context.Context, params domain.UploadFileParams) (*domain.File, error) {
					return nil, domain.ErrInvalidFilename
				},
			},
			log:  testLogger(),
			mapr: newFileMapper(testLogger()),
		}

		_, err := srv.UploadFile(context.Background(), &filev1.UploadFileRequest{})
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
	})

	It("uploads files", func() {
		svc := &fileSvcFake{
			createFilesFn: func(ctx context.Context, params domain.UploadFilesParams) ([]*domain.File, error) {
				Expect(params.OwnerID).To(Equal("owner-1"))
				Expect(params.Prefix).To(Equal("gallery"))
				Expect(params.Files).To(HaveLen(2))
				return []*domain.File{
					{
						ID:          "file-1",
						OwnerID:     "owner-1",
						Filename:    "one.png",
						Extension:   "png",
						ContentType: "image/png",
						Bucket:      "bucket",
						StoragePath: "gallery/file-1.png",
						SizeBytes:   3,
						CreatedAt:   time.Unix(1, 0).UTC(),
						UpdatedAt:   time.Unix(1, 0).UTC(),
					},
					{
						ID:          "file-2",
						OwnerID:     "owner-1",
						Filename:    "two",
						Extension:   "",
						ContentType: "text/plain",
						Bucket:      "bucket",
						StoragePath: "gallery/file-2",
						SizeBytes:   3,
						CreatedAt:   time.Unix(2, 0).UTC(),
						UpdatedAt:   time.Unix(2, 0).UTC(),
					},
				}, nil
			},
		}
		srv := &server{svc: svc, log: testLogger(), mapr: newFileMapper(testLogger())}

		resp, err := srv.UploadFiles(context.Background(), &filev1.UploadFilesRequest{
			OwnerId: "owner-1",
			Prefix:  "gallery",
			Files: []*filev1.UploadFileInput{
				{Filename: "one.png", ContentType: "image/png", Data: []byte("one")},
				{Filename: "two", ContentType: "text/plain", Data: []byte("two")},
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.GetFiles()).To(HaveLen(2))
		Expect(resp.GetFiles()[1].GetStoragePath()).To(Equal("gallery/file-2"))
		Expect(svc.createMany).To(HaveLen(1))
	})

	It("propagates upload files errors", func() {
		srv := &server{
			svc: &fileSvcFake{
				createFilesFn: func(ctx context.Context, params domain.UploadFilesParams) ([]*domain.File, error) {
					return nil, domain.ErrInvalidStoragePath
				},
			},
			log:  testLogger(),
			mapr: newFileMapper(testLogger()),
		}

		_, err := srv.UploadFiles(context.Background(), &filev1.UploadFilesRequest{})
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
	})

	It("gets and deletes files", func() {
		svc := &fileSvcFake{
			getFileFn: func(ctx context.Context, fileID string) (*domain.File, error) {
				if fileID == "missing" {
					return nil, domain.ErrFileNotFound
				}
				return &domain.File{
					ID:          fileID,
					OwnerID:     "owner-1",
					Filename:    "cover.png",
					Extension:   "png",
					ContentType: "image/png",
					Bucket:      "bucket",
					StoragePath: "gallery/" + fileID,
					SizeBytes:   3,
					CreatedAt:   time.Unix(1, 0).UTC(),
					UpdatedAt:   time.Unix(1, 0).UTC(),
				}, nil
			},
			deleteFileFn: func(ctx context.Context, fileID string) error {
				if fileID == "boom" {
					return errors.New("boom")
				}
				return nil
			},
		}
		srv := &server{svc: svc, log: testLogger(), mapr: newFileMapper(testLogger())}

		resp, err := srv.GetFile(context.Background(), &filev1.GetFileRequest{FileId: "file-1"})
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.GetFile().GetFileId()).To(Equal("file-1"))

		_, err = srv.GetFile(context.Background(), &filev1.GetFileRequest{FileId: "missing"})
		Expect(status.Code(err)).To(Equal(codes.NotFound))

		_, err = srv.DeleteFile(context.Background(), &filev1.DeleteFileRequest{FileId: "file-1"})
		Expect(err).NotTo(HaveOccurred())
		Expect(svc.deleteIDs).To(ContainElement("file-1"))

		_, err = srv.DeleteFile(context.Background(), &filev1.DeleteFileRequest{FileId: "boom"})
		Expect(status.Code(err)).To(Equal(codes.Internal))
	})

	It("gets file urls in batch", func() {
		srv := &server{
			svc: &fileSvcFake{
				getURLsFn: func(ctx context.Context, fileIDs []string) ([]domain.FileURL, error) {
					Expect(fileIDs).To(Equal([]string{"cover", "gallery"}))
					return []domain.FileURL{
						{ID: "cover", URL: "http://public.local/cover"},
						{ID: "gallery", URL: "http://public.local/gallery"},
					}, nil
				},
			},
			log:  testLogger(),
			mapr: newFileMapper(testLogger()),
		}

		resp, err := srv.GetFileURLs(context.Background(), &filev1.GetFileURLsRequest{FileIds: []string{"cover", "gallery"}})
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.GetFileUrls()).To(HaveLen(2))
		Expect(resp.GetFileUrls()[0].GetFileId()).To(Equal("cover"))
		Expect(resp.GetFileUrls()[1].GetUrl()).To(Equal("http://public.local/gallery"))
	})
})
