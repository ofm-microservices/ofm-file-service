package rustfs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Options groups the RustFS connection and object storage settings.
type Options struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Region    string
	Bucket    string
	Secure    bool
}

type s3API interface {
	HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
	CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

type s3PresignAPI interface {
	PresignPutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

// Storage exposes the object operations used by file-service.
type Storage interface {
	Put(ctx context.Context, objectKey, contentType string, data []byte) (int64, error)
	PresignPut(ctx context.Context, objectKey, contentType string) (string, error)
	PublicURL(objectKey string) string
	Exists(ctx context.Context, objectKey string) (bool, error)
	Delete(ctx context.Context, objectKey string) error
}
