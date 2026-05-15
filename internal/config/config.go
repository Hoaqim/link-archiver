package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr       string
	RedisAddr      string
	RedisQueueKey  string
	RedisDeadKey   string
	StorageBackend string
	StorageDir     string
	S3Bucket       string
	MaxAttempts    int
}

func Load() (Config, error) {
	attempts, err := envIntOr("MAX_ATTEMPTS", 3)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		HTTPAddr:       envOr("HTTP_SERVER_ADDR", ":8080"),
		RedisAddr:      envOr("REDIS_ADDR", "localhost:6379"),
		RedisQueueKey:  envOr("REDIS_QUEUE_KEY", "queue:jobs"),
		RedisDeadKey:   envOr("REDIS_DEAD_KEY", "queue:jobs:dead"),
		StorageBackend: envOr("STORAGE_BACKEND", "local"),
		StorageDir:     envOr("STORAGE_DIR", "./data"),
		S3Bucket:       os.Getenv("S3_BUCKET"),
		MaxAttempts:    attempts,
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	switch c.StorageBackend {
	case "local":
		if c.StorageDir == "" {
			return errors.New("STORAGE_DIR is required for local backend")
		}
	case "s3":
		if c.S3Bucket == "" {
			return errors.New("S3_BUCKET is required for s3 backend")
		}
	default:
		return fmt.Errorf("unknown STORAGE_BACKEND %q (want local or s3)", c.StorageBackend)
	}

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
