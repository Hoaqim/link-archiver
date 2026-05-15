package config

import (
	"fmt"
	"os"
)

type Config struct {
	HTTPAddr    string
	SQSQueueURL string
	S3Bucket    string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:    envOr("HTTP_SERVER_ADDR", ":8080"),
		SQSQueueURL: os.Getenv("SQS_URL"),
		S3Bucket:    os.Getenv("S3_BUCKET"),
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	var missing []string
	if c.SQSQueueURL == "" {
		missing = append(missing, "SQS_QUEUE_URL")
	}
	if c.S3Bucket == "" {
		missing = append(missing, "S3_BUCKET")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required env: %v", missing)
	}
	return nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
