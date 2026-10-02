package config

// KafkaConfig defines the file-service migration recovery transport boundary.
type KafkaConfig struct {
	Brokers                []string `env:"KAFKA_BROKERS" envSeparator:"," envDefault:"127.0.0.1:9092"`
	RecoveryTopic          string   `env:"KAFKA_FILE_RECOVERY_TOPIC" envDefault:"migration.recovery.commands.file"`
	RecoveryGroup          string   `env:"KAFKA_FILE_RECOVERY_GROUP" envDefault:"file-service-recovery"`
	RecoveryCompletedTopic string   `env:"KAFKA_FILE_RECOVERY_COMPLETED_TOPIC" envDefault:"migration.recovery.completed"`
	DeadLetterTopic        string   `env:"KAFKA_FILE_DLQ_TOPIC" envDefault:"file-service-dead-letter"`
}
