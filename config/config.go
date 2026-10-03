package config

import (
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config groups the full file-service runtime configuration.
type Config struct {
	App     AppConfig
	GRPC    GRPCConfig
	Metrics MetricsConfig
	Tracing TracingConfig
	DB      DBConfig `envPrefix:"DB_"`
	RustFS  RustFSConfig
	Kafka   KafkaConfig
}

// Load reads environment variables into Config and applies defaults declared
// on the individual config fields.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, WrapParseEnvConfigError(err)
	}
	if host := os.Getenv("DB_HOST"); host != "" {
		cfg.DB.Host = host
	}

	return cfg, nil
}
