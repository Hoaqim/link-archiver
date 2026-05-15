package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr    string
	SQSQueueURL string
	S3Bucket    string
	MaxAttempts int
}

func Load() (Config, error) {
	attempts, err := envIntOr("MAX_ATTEMPTS", 3)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		HTTPAddr:    envOr("HTTP_SERVER_ADDR", ":8080"),
		SQSQueueURL: os.Getenv("SQS_URL"),
		S3Bucket:    os.Getenv("S3_BUCKET"),
		MaxAttempts: attempts,
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	if c.MaxAttempts < 1 {
		return fmt.Errorf("MAX_ATTEMPTS must be >= 1, got %d", c.MaxAttempts)
	}
	return nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: %w", key, v, err)
	}
	return n, nil
}
