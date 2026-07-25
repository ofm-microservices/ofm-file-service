package rustfs

import "fmt"

// WrapLoadConfigError annotates S3 client configuration failures.
func WrapLoadConfigError(err error) error {
	return fmt.Errorf("load rustfs config: %w", err)
}

// WrapEnsureBucketError annotates bucket creation failures.
func WrapEnsureBucketError(bucket string, err error) error {
	return fmt.Errorf("ensure bucket %s: %w", bucket, err)
}

// WrapPutObjectError annotates object upload failures.
func WrapPutObjectError(objectKey string, err error) error {
	return fmt.Errorf("put object %s: %w", objectKey, err)
}

// WrapPresignPutObjectError annotates direct upload URL generation failures.
func WrapPresignPutObjectError(objectKey string, err error) error {
	return fmt.Errorf("presign put object %s: %w", objectKey, err)
}

// WrapDeleteObjectError annotates object deletion failures.
func WrapDeleteObjectError(objectKey string, err error) error {
	return fmt.Errorf("delete object %s: %w", objectKey, err)
}

// WrapHeadObjectError annotates object existence lookup failures.
func WrapHeadObjectError(objectKey string, err error) error {
	return fmt.Errorf("head object %s: %w", objectKey, err)
}
