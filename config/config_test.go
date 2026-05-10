package config

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Load", func() {
	keys := []string{
		"APP_ENV",
		"LOG_LEVEL",
		"GRPC_HOST",
		"GRPC_PORT",
		"SCYLLA_HOSTS",
		"SCYLLA_PORT",
		"SCYLLA_KEYSPACE",
		"SCYLLA_USERNAME",
		"SCYLLA_PASSWORD",
		"SCYLLA_CONSISTENCY",
		"SCYLLA_CONNECT_TIMEOUT",
		"SCYLLA_MAX_WAIT_SCHEMA_AGREEMENT",
		"SCYLLA_RETRY_ATTEMPTS",
		"SCYLLA_RETRY_BACKOFF",
		"RUSTFS_ENDPOINT",
		"RUSTFS_ACCESS_KEY",
		"RUSTFS_SECRET_KEY",
		"RUSTFS_REGION",
		"RUSTFS_BUCKET",
		"RUSTFS_SECURE",
	}

	unset := func() {
		for _, key := range keys {
			_ = os.Unsetenv(key)
		}
	}

	BeforeEach(func() {
		unset()
	})

	AfterEach(func() {
		unset()
	})

	It("loads defaults when no environment variables are set", func() {
		Expect(os.Setenv("RUSTFS_ENDPOINT", "http://localhost:9006")).To(Succeed())

		cfg, err := Load()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).To(Equal(&Config{
			App: AppConfig{
				Env:      "local",
				LogLevel: "info",
			},
			GRPC: GRPCConfig{
				Host: "0.0.0.0",
				Port: 9504,
			},
			Scylla: ScyllaConfig{
				Hosts:                  []string{"127.0.0.1"},
				Port:                   9042,
				Keyspace:               "file_service",
				Consistency:            "quorum",
				ConnectTimeout:         10 * time.Second,
				MaxWaitSchemaAgreement: 30 * time.Second,
				RetryAttempts:          20,
				RetryBackoff:           2 * time.Second,
			},
			RustFS: RustFSConfig{
				Endpoint:  "http://localhost:9006",
				AccessKey: "rustfsadmin",
				SecretKey: "rustfsadmin",
				Region:    "us-east-1",
				Bucket:    "ofm-files",
				Secure:    false,
			},
		}))
	})

	It("parses environment variables into the config struct", func() {
		Expect(os.Setenv("APP_ENV", "test")).To(Succeed())
		Expect(os.Setenv("LOG_LEVEL", "debug")).To(Succeed())
		Expect(os.Setenv("GRPC_HOST", "127.0.0.1")).To(Succeed())
		Expect(os.Setenv("GRPC_PORT", "9504")).To(Succeed())
		Expect(os.Setenv("SCYLLA_HOSTS", "scylla-1,scylla-2")).To(Succeed())
		Expect(os.Setenv("SCYLLA_PORT", "19042")).To(Succeed())
		Expect(os.Setenv("SCYLLA_KEYSPACE", "files_test")).To(Succeed())
		Expect(os.Setenv("SCYLLA_USERNAME", "admin")).To(Succeed())
		Expect(os.Setenv("SCYLLA_PASSWORD", "secret")).To(Succeed())
		Expect(os.Setenv("SCYLLA_CONSISTENCY", "one")).To(Succeed())
		Expect(os.Setenv("SCYLLA_CONNECT_TIMEOUT", "15s")).To(Succeed())
		Expect(os.Setenv("SCYLLA_MAX_WAIT_SCHEMA_AGREEMENT", "45s")).To(Succeed())
		Expect(os.Setenv("SCYLLA_RETRY_ATTEMPTS", "3")).To(Succeed())
		Expect(os.Setenv("SCYLLA_RETRY_BACKOFF", "250ms")).To(Succeed())
		Expect(os.Setenv("RUSTFS_ENDPOINT", "http://localhost:9006")).To(Succeed())
		Expect(os.Setenv("RUSTFS_ACCESS_KEY", "ak")).To(Succeed())
		Expect(os.Setenv("RUSTFS_SECRET_KEY", "sk")).To(Succeed())
		Expect(os.Setenv("RUSTFS_REGION", "eu-west-1")).To(Succeed())
		Expect(os.Setenv("RUSTFS_BUCKET", "files")).To(Succeed())
		Expect(os.Setenv("RUSTFS_SECURE", "true")).To(Succeed())

		cfg, err := Load()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.App.Env).To(Equal("test"))
		Expect(cfg.App.LogLevel).To(Equal("debug"))
		Expect(cfg.GRPC.Host).To(Equal("127.0.0.1"))
		Expect(cfg.GRPC.Port).To(Equal(9504))
		Expect(cfg.Scylla.Hosts).To(Equal([]string{"scylla-1", "scylla-2"}))
		Expect(cfg.Scylla.Port).To(Equal(19042))
		Expect(cfg.Scylla.Keyspace).To(Equal("files_test"))
		Expect(cfg.Scylla.Username).To(Equal("admin"))
		Expect(cfg.Scylla.Password).To(Equal("secret"))
		Expect(cfg.Scylla.Consistency).To(Equal("one"))
		Expect(cfg.Scylla.ConnectTimeout).To(Equal(15 * time.Second))
		Expect(cfg.Scylla.MaxWaitSchemaAgreement).To(Equal(45 * time.Second))
		Expect(cfg.Scylla.RetryAttempts).To(Equal(3))
		Expect(cfg.Scylla.RetryBackoff).To(Equal(250 * time.Millisecond))
		Expect(cfg.RustFS.Endpoint).To(Equal("http://localhost:9006"))
		Expect(cfg.RustFS.AccessKey).To(Equal("ak"))
		Expect(cfg.RustFS.SecretKey).To(Equal("sk"))
		Expect(cfg.RustFS.Region).To(Equal("eu-west-1"))
		Expect(cfg.RustFS.Bucket).To(Equal("files"))
		Expect(cfg.RustFS.Secure).To(BeTrue())
	})

	It("wraps parse errors", func() {
		Expect(os.Setenv("RUSTFS_ENDPOINT", "http://localhost:9006")).To(Succeed())
		Expect(os.Setenv("GRPC_PORT", "not-a-number")).To(Succeed())

		cfg, err := Load()
		Expect(cfg).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("parse env config"))
	})
})
