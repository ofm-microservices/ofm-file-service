package mapper

import (
	"time"

	"file-service/internal/domain"
	"file-service/internal/infra/write/scylla/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("mapping", func() {
	It("maps domain files to rows and back", func() {
		now := time.Unix(123, 456).UTC()
		file := domain.File{
			ID:          "file-1",
			OwnerID:     "owner-1",
			Filename:    "cover.png",
			Extension:   "png",
			ContentType: "image/png",
			Bucket:      "bucket",
			StoragePath: "gallery/file-1.png",
			SizeBytes:   123,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		row := MapDomainFileToRow(file)
		Expect(row).To(Equal(model.FileRow{
			ID:          "file-1",
			OwnerID:     "owner-1",
			Filename:    "cover.png",
			Extension:   "png",
			ContentType: "image/png",
			Bucket:      "bucket",
			StoragePath: "gallery/file-1.png",
			SizeBytes:   123,
			CreatedAt:   now,
			UpdatedAt:   now,
		}))

		mapped := MapRowToDomainFile(row)
		Expect(mapped).To(Equal(&file))
	})
})
