package rustfs

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fakeS3 struct {
	headErr    error
	createErr  error
	putErr     error
	deleteErr  error
	headCalls   int
	createCalls int
	putCalls    int
	deleteCalls int
	mu         sync.Mutex
}

func (f *fakeS3) HeadBucket(context.Context, *s3.HeadBucketInput, ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.headCalls++
	return &s3.HeadBucketOutput{}, f.headErr
}

func (f *fakeS3) CreateBucket(context.Context, *s3.CreateBucketInput, ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createCalls++
	return &s3.CreateBucketOutput{}, f.createErr
}

func (f *fakeS3) PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.putCalls++
	return &s3.PutObjectOutput{}, f.putErr
}

func (f *fakeS3) DeleteObject(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleteCalls++
	return &s3.DeleteObjectOutput{}, f.deleteErr
}

var _ = Describe("wrappers", func() {
	It("wraps errors", func() {
		Expect(WrapLoadConfigError(errors.New("boom")).Error()).To(ContainSubstring("load rustfs config"))
		Expect(WrapEnsureBucketError("bucket", errors.New("boom")).Error()).To(ContainSubstring("ensure bucket bucket"))
		Expect(WrapPutObjectError("key", errors.New("boom")).Error()).To(ContainSubstring("put object key"))
		Expect(WrapDeleteObjectError("key", errors.New("boom")).Error()).To(ContainSubstring("delete object key"))
	})
})

var _ = Describe("Open", func() {
	var (
		origLoadDefaultConfig func(context.Context, ...func(*awsconfig.LoadOptions) error) (aws.Config, error)
		origNewS3Client       func(aws.Config, ...func(*s3.Options)) s3API
		origSleep             func(time.Duration)
	)

	BeforeEach(func() {
		origLoadDefaultConfig = loadDefaultConfig
		origNewS3Client = newS3Client
		origSleep = sleep
		loadDefaultConfig = func(context.Context, ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
			return aws.Config{}, nil
		}
		sleep = func(time.Duration) {}
	})

	AfterEach(func() {
		loadDefaultConfig = origLoadDefaultConfig
		newS3Client = origNewS3Client
		sleep = origSleep
	})

	It("wraps config loading failures", func() {
		loadDefaultConfig = func(context.Context, ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
			return aws.Config{}, errors.New("config failed")
		}

		storage, err := Open(context.Background(), Options{
			Endpoint:  "http://example.invalid",
			AccessKey: "ak",
			SecretKey: "sk",
			Region:    "us-east-1",
			Bucket:    "bucket",
		})
		Expect(storage).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("load rustfs config"))
	})

	It("opens an existing bucket and performs object operations", func() {
		fake := &fakeS3{}
		newS3Client = func(aws.Config, ...func(*s3.Options)) s3API {
			return fake
		}

		storage, err := Open(context.Background(), Options{
			Endpoint:  "http://example.invalid",
			AccessKey: "ak",
			SecretKey: "sk",
			Region:    "us-east-1",
			Bucket:    "bucket",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(storage).NotTo(BeNil())
		Expect(storage.(*client).String()).To(Equal("rustfs(bucket)"))

		size, err := storage.Put(context.Background(), "path/file.txt", "text/plain", []byte("hello"))
		Expect(err).NotTo(HaveOccurred())
		Expect(size).To(Equal(int64(5)))

		Expect(storage.Delete(context.Background(), "path/file.txt")).To(Succeed())
		Expect(fake.headCalls).To(Equal(1))
		Expect(fake.createCalls).To(Equal(0))
		Expect(fake.putCalls).To(Equal(1))
		Expect(fake.deleteCalls).To(Equal(1))
	})

	It("creates the bucket when it is missing", func() {
		fake := &fakeS3{headErr: errors.New("missing")}
		newS3Client = func(aws.Config, ...func(*s3.Options)) s3API {
			return fake
		}

		storage, err := Open(context.Background(), Options{
			Endpoint:  "http://example.invalid",
			AccessKey: "ak",
			SecretKey: "sk",
			Region:    "us-east-1",
			Bucket:    "bucket",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(storage).NotTo(BeNil())
		Expect(fake.headCalls).To(Equal(1))
		Expect(fake.createCalls).To(Equal(1))
	})

	It("returns wrapped errors for object operations", func() {
		fake := &fakeS3{putErr: errors.New("boom"), deleteErr: errors.New("boom")}
		newS3Client = func(aws.Config, ...func(*s3.Options)) s3API {
			return fake
		}

		storage, err := Open(context.Background(), Options{
			Endpoint:  "http://example.invalid",
			AccessKey: "ak",
			SecretKey: "sk",
			Region:    "us-east-1",
			Bucket:    "bucket",
		})
		Expect(err).NotTo(HaveOccurred())

		_, err = storage.Put(context.Background(), "key", "text/plain", []byte("hello"))
		Expect(err).To(MatchError(ContainSubstring("put object key")))

		err = storage.Delete(context.Background(), "key")
		Expect(err).To(MatchError(ContainSubstring("delete object key")))
	})

	It("wraps bucket setup failures", func() {
		fake := &fakeS3{headErr: errors.New("missing"), createErr: errors.New("boom")}
		newS3Client = func(aws.Config, ...func(*s3.Options)) s3API {
			return fake
		}

		storage, err := Open(context.Background(), Options{
			Endpoint:  "http://example.invalid",
			AccessKey: "ak",
			SecretKey: "sk",
			Region:    "us-east-1",
			Bucket:    "bucket",
		})
		Expect(storage).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("ensure bucket bucket"))
	})
})

var _ = Describe("client", func() {
	It("formats the storage string", func() {
		Expect((&client{bucket: "bucket"}).String()).To(Equal("rustfs(bucket)"))
	})
})
