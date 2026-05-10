package config

// RustFSConfig defines the RustFS S3-compatible object storage settings.
type RustFSConfig struct {
	Endpoint  string `env:"RUSTFS_ENDPOINT,required"`
	AccessKey string `env:"RUSTFS_ACCESS_KEY" envDefault:"rustfsadmin"`
	SecretKey string `env:"RUSTFS_SECRET_KEY" envDefault:"rustfsadmin"`
	Region    string `env:"RUSTFS_REGION" envDefault:"us-east-1"`
	Bucket    string `env:"RUSTFS_BUCKET" envDefault:"ofm-files"`
	Secure    bool   `env:"RUSTFS_SECURE" envDefault:"false"`
}
