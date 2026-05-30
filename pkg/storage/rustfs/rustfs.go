package rustfs

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ofm-microservices/ofm-common/pkg/observability/metrics"
)

type client struct {
	bucket string
	s3     s3API
	endpoint string
}

var loadDefaultConfig = awsconfig.LoadDefaultConfig

var newS3Client = func(cfg aws.Config, optFns ...func(*s3.Options)) s3API {
	return s3.NewFromConfig(cfg, optFns...)
}

var sleep = time.Sleep

// Open creates a RustFS client, verifies the bucket exists, and returns the
// object storage adapter.
func Open(ctx context.Context, cfg Options) (Storage, error) {
	awsCfg, err := loadDefaultConfig(
		ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	)
	if err != nil {
		return nil, WrapLoadConfigError(err)
	}

	s3Client := newS3Client(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(cfg.Endpoint)
	})

	c := &client{bucket: cfg.Bucket, s3: s3Client, endpoint: cfg.Endpoint}
	var lastErr error
	for i := 0; i < 10; i++ {
		if _, err := c.s3.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(cfg.Bucket)}); err == nil {
			return c, nil
		} else {
			lastErr = err
			if _, err := c.s3.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(cfg.Bucket)}); err == nil {
				return c, nil
			} else {
				lastErr = err
			}
		}
		sleep(1 * time.Second)
	}

	return nil, WrapEnsureBucketError(cfg.Bucket, lastErr)
}

func (c *client) Put(ctx context.Context, objectKey, contentType string, data []byte) (int64, error) {
	started := time.Now()
	status := "success"
	defer func() { metrics.Global().ObserveObjectStorage("put", c.bucket, status, time.Since(started)) }()

	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(objectKey),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		status = "error"
		return 0, WrapPutObjectError(objectKey, err)
	}

	return int64(len(data)), nil
}

func (c *client) PresignPut(ctx context.Context, objectKey, contentType string) (string, error) {
	_ = ctx
	_ = contentType
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(c.endpoint, "/"), c.bucket, objectKey), nil
}

func (c *client) PublicURL(objectKey string) string {
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(c.endpoint, "/"), c.bucket, objectKey)
}

func (c *client) Delete(ctx context.Context, objectKey string) error {
	started := time.Now()
	status := "success"
	defer func() { metrics.Global().ObserveObjectStorage("delete", c.bucket, status, time.Since(started)) }()

	if _, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(objectKey),
	}); err != nil {
		status = "error"
		return WrapDeleteObjectError(objectKey, err)
	}

	return nil
}

func (c *client) String() string {
	return fmt.Sprintf("rustfs(%s)", c.bucket)
}
