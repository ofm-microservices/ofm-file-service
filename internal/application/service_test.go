package application

import (
	"context"
	"errors"
	"path/filepath"

	"file-service/internal/domain"
	"github.com/google/uuid"
	"github.com/ofm-microservices/ofm-common/pkg/logging"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type repoFake struct {
	createFn   func(context.Context, domain.File) (*domain.File, error)
	getFn      func(context.Context, string) (*domain.File, error)
	deleteFn   func(context.Context, string) error
	creates    []domain.File
	getIDs     []string
	deleteIDs  []string
}

func (r *repoFake) Create(ctx context.Context, file domain.File) (*domain.File, error) {
	r.creates = append(r.creates, file)
	if r.createFn != nil {
		return r.createFn(ctx, file)
	}
	f := file
	return &f, nil
}

func (r *repoFake) GetByID(ctx context.Context, fileID string) (*domain.File, error) {
	r.getIDs = append(r.getIDs, fileID)
	if r.getFn != nil {
		return r.getFn(ctx, fileID)
	}
	return nil, domain.ErrFileNotFound
}

func (r *repoFake) DeleteByID(ctx context.Context, fileID string) error {
	r.deleteIDs = append(r.deleteIDs, fileID)
	if r.deleteFn != nil {
		return r.deleteFn(ctx, fileID)
	}
	return nil
}

type storageCall struct {
	objectKey   string
	contentType string
	data        []byte
}

type storageFake struct {
	putFn     func(context.Context, string, string, []byte) (int64, error)
	deleteFn  func(context.Context, string) error
	putCalls   []storageCall
	deleteKeys []string
}

func (s *storageFake) PresignPut(ctx context.Context, objectKey, contentType string) (string, error) {
	return "http://upload.local/" + objectKey, nil
}

func (s *storageFake) PublicURL(objectKey string) string {
	return "http://public.local/" + objectKey
}

func (s *storageFake) Put(ctx context.Context, objectKey, contentType string, data []byte) (int64, error) {
	s.putCalls = append(s.putCalls, storageCall{objectKey: objectKey, contentType: contentType, data: append([]byte(nil), data...)})
	if s.putFn != nil {
		return s.putFn(ctx, objectKey, contentType, data)
	}
	return int64(len(data)), nil
}

func (s *storageFake) Delete(ctx context.Context, objectKey string) error {
	s.deleteKeys = append(s.deleteKeys, objectKey)
	if s.deleteFn != nil {
		return s.deleteFn(ctx, objectKey)
	}
	return nil
}

func newTestLogger() logging.Logger {
	lg, err := logging.New("file-service-test", "test", "error")
	Expect(err).NotTo(HaveOccurred())
	return lg
}

func newTestService(repo domain.FileRepository, storage domain.FileStorage, bucket string) FileService {
	svc, err := New(repo, storage, bucket, newTestLogger())
	Expect(err).NotTo(HaveOccurred())
	return svc
}

var _ = Describe("New", func() {
	It("rejects nil collaborators and empty bucket", func() {
		_, err := New(nil, &storageFake{}, "bucket", newTestLogger())
		Expect(err).To(MatchError(ErrNilFileRepository))

		_, err = New(&repoFake{}, nil, "bucket", newTestLogger())
		Expect(err).To(MatchError(ErrNilFileStorage))

		_, err = New(&repoFake{}, &storageFake{}, "   ", newTestLogger())
		Expect(err).To(MatchError(ErrEmptyBucket))

		_, err = New(&repoFake{}, &storageFake{}, "bucket", nil)
		Expect(err).To(MatchError(ErrNilLogger))
	})

	It("trims the bucket name", func() {
		svc := newTestService(&repoFake{}, &storageFake{}, "  ofm-files  ")
		Expect(svc).NotTo(BeNil())
	})
})

var _ = Describe("CreateFile", func() {
	It("delegates to CreateFiles and returns the first file", func() {
		repo := &repoFake{}
		storage := &storageFake{}
		svc := newTestService(repo, storage, "bucket")

		file, err := svc.CreateFile(context.Background(), domain.UploadFileParams{
			OwnerID:     "owner-1",
			Filename:    "hero.png",
			ContentType: "image/png",
			Prefix:      "/gallery/",
			Data:        []byte("abc"),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(file).NotTo(BeNil())
		Expect(file.OwnerID).To(Equal("owner-1"))
		Expect(file.Filename).To(Equal("hero.png"))
		Expect(file.Extension).To(Equal("png"))
		Expect(file.ContentType).To(Equal("image/png"))
		Expect(file.StoragePath).To(HavePrefix("gallery/"))
		Expect(file.StoragePath).To(HaveSuffix(".png"))
		Expect(file.SizeBytes).To(Equal(int64(3)))
		Expect(uuid.MustParse(file.ID).Version()).To(Equal(uuid.Version(7)))
		Expect(repo.creates).To(HaveLen(1))
		Expect(storage.putCalls).To(HaveLen(1))
	})
})

var _ = Describe("CreateFiles", func() {
	It("validates request data", func() {
		svc := newTestService(&repoFake{}, &storageFake{}, "bucket")

		_, err := svc.CreateFiles(context.Background(), domain.UploadFilesParams{})
		Expect(err).To(MatchError(domain.ErrInvalidOwnerID))

		_, err = svc.CreateFiles(context.Background(), domain.UploadFilesParams{OwnerID: "owner"})
		Expect(err).To(MatchError(domain.ErrInvalidStoragePath))

		_, err = svc.CreateFiles(context.Background(), domain.UploadFilesParams{OwnerID: "owner", Prefix: "gallery"})
		Expect(err).To(MatchError(domain.ErrInvalidFileData))
	})

	It("stores multiple files and builds keys with and without extensions", func() {
		repo := &repoFake{}
		storage := &storageFake{}
		svc := newTestService(repo, storage, "bucket")

		files, err := svc.CreateFiles(context.Background(), domain.UploadFilesParams{
			OwnerID: "owner-1",
			Prefix:  "/gallery/",
			Files: []domain.UploadFileParams{
				{Filename: "cover.JPG", ContentType: "image/jpeg", Data: []byte("abc")},
				{Filename: "README", ContentType: "text/plain", Data: []byte("xyz")},
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(files).To(HaveLen(2))
		Expect(storage.putCalls).To(HaveLen(2))
		Expect(storage.putCalls[0].objectKey).To(HavePrefix("gallery/"))
		Expect(storage.putCalls[0].objectKey).To(HaveSuffix(".jpg"))
		Expect(storage.putCalls[1].objectKey).To(HavePrefix("gallery/"))
		Expect(filepath.Ext(storage.putCalls[1].objectKey)).To(BeEmpty())
		Expect(repo.creates).To(HaveLen(2))
		Expect(files[0].Bucket).To(Equal("bucket"))
		Expect(files[1].Bucket).To(Equal("bucket"))
	})

	It("compensates previously created files when storage fails", func() {
		repo := &repoFake{}
		var putCount int
		storage := &storageFake{
			putFn: func(ctx context.Context, objectKey, contentType string, data []byte) (int64, error) {
				putCount++
				if putCount == 1 {
					return int64(len(data)), nil
				}
				return 0, errors.New("store failed")
			},
		}
		svc := newTestService(repo, storage, "bucket")

		_, err := svc.CreateFiles(context.Background(), domain.UploadFilesParams{
			OwnerID: "owner",
			Prefix:  "gallery",
			Files: []domain.UploadFileParams{
				{Filename: "one.png", ContentType: "image/png", Data: []byte("one")},
				{Filename: "two.png", ContentType: "image/png", Data: []byte("two")},
			},
		})
		Expect(err).To(MatchError(domain.ErrFailedToStoreFile))
		Expect(repo.deleteIDs).To(HaveLen(1))
		Expect(storage.deleteKeys).To(HaveLen(1))
	})

	It("cleans up the stored object when metadata persistence fails", func() {
		var createCount int
		repo := &repoFake{
			createFn: func(ctx context.Context, file domain.File) (*domain.File, error) {
				createCount++
				if createCount == 1 {
					f := file
					return &f, nil
				}
				return nil, errors.New("create failed")
			},
		}
		storage := &storageFake{}
		svc := newTestService(repo, storage, "bucket")

		_, err := svc.CreateFiles(context.Background(), domain.UploadFilesParams{
			OwnerID: "owner",
			Prefix:  "gallery",
			Files: []domain.UploadFileParams{
				{Filename: "one.png", ContentType: "image/png", Data: []byte("one")},
				{Filename: "two.png", ContentType: "image/png", Data: []byte("two")},
			},
		})
		Expect(err).To(MatchError("create failed"))
		Expect(storage.deleteKeys).To(HaveLen(2))
		Expect(repo.deleteIDs).To(HaveLen(1))
	})

	It("skips nil files during compensation", func() {
		repo := &repoFake{}
		storage := &storageFake{}
		svc := newTestService(repo, storage, "bucket").(*fileService)

		svc.compensateCreatedFiles([]*domain.File{
			nil,
			&domain.File{ID: "file-1", StoragePath: "gallery/file-1"},
		})

		Expect(repo.deleteIDs).To(Equal([]string{"file-1"}))
		Expect(storage.deleteKeys).To(Equal([]string{"gallery/file-1"}))
	})
})

var _ = Describe("GetFile", func() {
	It("validates the file id and delegates lookup", func() {
		repo := &repoFake{
			getFn: func(ctx context.Context, fileID string) (*domain.File, error) {
				return &domain.File{ID: fileID, StoragePath: "gallery/" + fileID}, nil
			},
		}
		svc := newTestService(repo, &storageFake{}, "bucket")

		_, err := svc.GetFile(context.Background(), "  ")
		Expect(err).To(MatchError(domain.ErrInvalidFileID))

		file, err := svc.GetFile(context.Background(), "file-1")
		Expect(err).NotTo(HaveOccurred())
		Expect(file.ID).To(Equal("file-1"))
		Expect(repo.getIDs).To(Equal([]string{"file-1"}))
	})
})

var _ = Describe("DeleteFile", func() {
	It("validates the file id", func() {
		svc := newTestService(&repoFake{}, &storageFake{}, "bucket")

		err := svc.DeleteFile(context.Background(), " ")
		Expect(err).To(MatchError(domain.ErrInvalidFileID))
	})

	It("returns lookup errors and storage delete errors", func() {
		repo := &repoFake{
			getFn: func(ctx context.Context, fileID string) (*domain.File, error) {
				switch fileID {
				case "missing":
					return nil, domain.ErrFileNotFound
				case "storage-fail":
					return &domain.File{ID: fileID, StoragePath: "gallery/" + fileID}, nil
				case "repo-fail":
					return &domain.File{ID: fileID, StoragePath: "gallery/" + fileID}, nil
				default:
					return &domain.File{ID: fileID, StoragePath: "gallery/" + fileID}, nil
				}
			},
			deleteFn: func(ctx context.Context, fileID string) error {
				if fileID == "repo-fail" {
					return errors.New("delete failed")
				}
				return nil
			},
		}
		storage := &storageFake{
			deleteFn: func(ctx context.Context, objectKey string) error {
				if objectKey == "gallery/storage-fail" {
					return errors.New("storage delete failed")
				}
				return nil
			},
		}
		svc := newTestService(repo, storage, "bucket")

		err := svc.DeleteFile(context.Background(), "missing")
		Expect(err).To(MatchError(domain.ErrFileNotFound))

		err = svc.DeleteFile(context.Background(), "storage-fail")
		Expect(err).To(MatchError(domain.ErrFailedToDeleteStoredFile))

		err = svc.DeleteFile(context.Background(), "repo-fail")
		Expect(err).To(MatchError("delete failed"))
	})

	It("deletes stored data and metadata", func() {
		repo := &repoFake{
			getFn: func(ctx context.Context, fileID string) (*domain.File, error) {
				return &domain.File{ID: fileID, StoragePath: "gallery/" + fileID}, nil
			},
		}
		storage := &storageFake{}
		svc := newTestService(repo, storage, "bucket")

		err := svc.DeleteFile(context.Background(), "file-1")
		Expect(err).NotTo(HaveOccurred())
		Expect(storage.deleteKeys).To(Equal([]string{"gallery/file-1"}))
		Expect(repo.deleteIDs).To(Equal([]string{"file-1"}))
	})
})
