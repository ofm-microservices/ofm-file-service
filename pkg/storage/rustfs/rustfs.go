package rustfs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/ofm-microservices/ofm-common/pkg/observability/metrics"
	"github.com/ofm-microservices/ofm-common/pkg/resilience"
)

type client struct {
	bucket    string
	s3        s3API
	presigner s3PresignAPI
	endpoint  string
	breaker   *resilience.Breaker
}

var loadDefaultConfig = awsconfig.LoadDefaultConfig

var newS3Client = func(cfg aws.Config, optFns ...func(*s3.Options)) s3API {
	return s3.NewFromConfig(cfg, optFns...)
}

var newS3Presigner = func(cfg aws.Config, optFns ...func(*s3.Options)) s3PresignAPI {
	return s3.NewPresignClient(s3.NewFromConfig(cfg, optFns...))
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

	s3Options := func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(cfg.Endpoint)
	}

	c := &client{
		bucket:    cfg.Bucket,
		s3:        newS3Client(awsCfg, s3Options),
		presigner: newS3Presigner(awsCfg, s3Options),
		endpoint:  cfg.Endpoint,
		breaker:   resilience.NewBreaker(resilience.BreakerConfigFromEnv()),
	}
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
	c.ensureBreaker()
	started := time.Now()
	status := "success"
	defer func() { metrics.Global().ObserveObjectStorage("put", c.bucket, status, time.Since(started)) }()

	err := c.breaker.Do(ctx, func(callCtx context.Context) error {
		_, err := c.s3.PutObject(callCtx, &s3.PutObjectInput{Bucket: aws.String(c.bucket), Key: aws.String(objectKey), Body: bytes.NewReader(data), ContentType: aws.String(contentType)})
		return err
	})
	if err != nil {
		status = "error"
		return 0, WrapPutObjectError(objectKey, err)
	}

	return int64(len(data)), nil
}

func (c *client) PresignPut(ctx context.Context, objectKey, contentType string) (string, error) {
	c.ensureBreaker()
	var req *v4.PresignedHTTPRequest
	err := c.breaker.Do(ctx, func(callCtx context.Context) error {
		var err error
		req, err = c.presigner.PresignPutObject(callCtx, &s3.PutObjectInput{Bucket: aws.String(c.bucket), Key: aws.String(objectKey), ContentType: aws.String(contentType)})
		return err
	})
	if err != nil {
		return "", WrapPresignPutObjectError(objectKey, err)
	}
	return req.URL, nil
}

func (c *client) PublicURL(objectKey string) string {
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(c.endpoint, "/"), c.bucket, objectKey)
}

func (c *client) Exists(ctx context.Context, objectKey string) (bool, error) {
	c.ensureBreaker()
	err := c.breaker.Do(ctx, func(callCtx context.Context) error {
		_, err := c.s3.HeadObject(callCtx, &s3.HeadObjectInput{Bucket: aws.String(c.bucket), Key: aws.String(objectKey)})
		return err
	})
	if err == nil {
		return true, nil
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NotFound" {
		return false, nil
	}
	return false, WrapHeadObjectError(objectKey, err)
}

func (c *client) Delete(ctx context.Context, objectKey string) error {
	c.ensureBreaker()
	started := time.Now()
	status := "success"
	defer func() { metrics.Global().ObserveObjectStorage("delete", c.bucket, status, time.Since(started)) }()

	if err := c.breaker.Do(ctx, func(callCtx context.Context) error {
		_, err := c.s3.DeleteObject(callCtx, &s3.DeleteObjectInput{Bucket: aws.String(c.bucket), Key: aws.String(objectKey)})
		return err
	}); err != nil {
		status = "error"
		return WrapDeleteObjectError(objectKey, err)
	}

	return nil
}

func (c *client) String() string {
	return fmt.Sprintf("rustfs(%s)", c.bucket)
}

func (c *client) ensureBreaker() {
	if c.breaker == nil {
		c.breaker = resilience.NewBreaker(resilience.BreakerConfigFromEnv())
	}
}
