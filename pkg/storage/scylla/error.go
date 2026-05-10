package scylla

import "fmt"

// WrapCreateClusterSessionError annotates Scylla cluster connection failures.
func WrapCreateClusterSessionError(err error) error {
	return fmt.Errorf("create cluster session: %w", err)
}

// WrapEnsureSchemaError annotates Scylla schema creation failures.
func WrapEnsureSchemaError(err error) error {
	return fmt.Errorf("ensure schema: %w", err)
}

// WrapOpenSessionError annotates Scylla session open failures.
func WrapOpenSessionError(err error) error {
	return fmt.Errorf("open scylla session: %w", err)
}
