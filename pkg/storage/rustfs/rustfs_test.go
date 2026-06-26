package rustfs

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fakeS3 struct {
	headErr      error
	createErr    error
	putErr       error
	presignErr   error
	deleteErr    error
	headObjErr   error
	headCalls    int
	createCalls  int
	putCalls     int
	presignCalls int
	deleteCalls  int
	headObjCalls int
	mu           sync.Mutex
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

func (f *fakeS3) HeadObject(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.headObjCalls++
	return &s3.HeadObjectOutput{}, f.headObjErr
}

func (f *fakeS3) PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.putCalls++
	return &s3.PutObjectOutput{}, f.putErr
}

func (f *fakeS3) PresignPutObject(context.Context, *s3.PutObjectInput, ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.presignCalls++
	return &v4.PresignedHTTPRequest{URL: "http://rustfs.local/bucket/key"}, f.presignErr
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
		Expect(WrapPresignPutObjectError("key", errors.New("boom")).Error()).To(ContainSubstring("presign put object key"))
		Expect(WrapDeleteObjectError("key", errors.New("boom")).Error()).To(ContainSubstring("delete object key"))
	})
})

var _ = Describe("Open", func() {
	var (
		origLoadDefaultConfig func(context.Context, ...func(*awsconfig.LoadOptions) error) (aws.Config, error)
		origNewS3Client       func(aws.Config, ...func(*s3.Options)) s3API
		origNewS3Presigner    func(aws.Config, ...func(*s3.Options)) s3PresignAPI
		origSleep             func(time.Duration)
	)

	BeforeEach(func() {
		origLoadDefaultConfig = loadDefaultConfig
		origNewS3Client = newS3Client
		origNewS3Presigner = newS3Presigner
		origSleep = sleep
		loadDefaultConfig = func(context.Context, ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
			return aws.Config{}, nil
		}
		newS3Presigner = func(aws.Config, ...func(*s3.Options)) s3PresignAPI {
			return &fakeS3{}
		}
		sleep = func(time.Duration) {}
	})

	AfterEach(func() {
		loadDefaultConfig = origLoadDefaultConfig
		newS3Client = origNewS3Client
		newS3Presigner = origNewS3Presigner
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
		newS3Presigner = func(aws.Config, ...func(*s3.Options)) s3PresignAPI {
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

	It("generates real presigned direct upload URLs", func() {
		fake := &fakeS3{}
		newS3Client = func(aws.Config, ...func(*s3.Options)) s3API {
			return fake
		}
		newS3Presigner = func(aws.Config, ...func(*s3.Options)) s3PresignAPI {
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

		uploadURL, err := storage.PresignPut(context.Background(), "path/file.txt", "text/plain")
		Expect(err).NotTo(HaveOccurred())
		Expect(uploadURL).To(Equal("http://rustfs.local/bucket/key"))
		Expect(fake.presignCalls).To(Equal(1))
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
		fake := &fakeS3{putErr: errors.New("boom"), presignErr: errors.New("boom"), deleteErr: errors.New("boom")}
		newS3Client = func(aws.Config, ...func(*s3.Options)) s3API {
			return fake
		}
		newS3Presigner = func(aws.Config, ...func(*s3.Options)) s3PresignAPI {
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

		_, err = storage.PresignPut(context.Background(), "key", "text/plain")
		Expect(err).To(MatchError(ContainSubstring("presign put object key")))

		err = storage.Delete(context.Background(), "key")
		Expect(err).To(MatchError(ContainSubstring("delete object key")))
	})

	It("checks whether an object exists", func() {
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

		exists, err := storage.Exists(context.Background(), "key")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeTrue())
		Expect(fake.headObjCalls).To(Equal(1))
	})

	It("treats missing objects as not ready", func() {
		fake := &fakeS3{headObjErr: &smithy.GenericAPIError{Code: "NotFound", Message: "missing"}}
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

		exists, err := storage.Exists(context.Background(), "key")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeFalse())
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
