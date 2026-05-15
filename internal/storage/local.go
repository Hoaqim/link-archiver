package storage

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type Local struct {
	baseDir string
}

func NewLocal(baseDir string) (*Local, error) {
	if err := os.MkdirAll(baseDir, 0o777); err != nil {
		return nil, fmt.Errorf("creating base dir: %w", err)
	}
	return &Local{baseDir: baseDir}, nil
}

func (d *Local) path(key string) string {
	return filepath.Join(d.baseDir, key)
}

func (d *Local) Put(ctx context.Context, key string, data []byte, contentType string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := os.WriteFile(d.path(key), data, 0o644); err != nil {
		return fmt.Errorf("write file error %w", err)
	}

	if err := os.WriteFile(d.path(key)+".ct", []byte(contentType), 0o644); err != nil {
		return fmt.Errorf("write content-type sidecar: %w", err)
	}

	return nil
}

func (d *Local) Get(ctx context.Context, key string) ([]byte, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}

	data, err := os.ReadFile(d.path(key))
	if err != nil {
		return nil, "", fmt.Errorf("read file %w", err)
	}

	ct, err := os.ReadFile(d.path(key) + ".ct")
	if err != nil {
		return nil, "", fmt.Errorf("read content-type %w", err)
	}

	return data, string(ct), nil
}

func (d *Local) Exists(ctx context.Context, key string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	_, err := os.Stat(d.path(key))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
