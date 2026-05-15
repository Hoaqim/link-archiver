package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := Config{
		HTTPAddr:    ":8080",
		S3Bucket:    "",
		MaxAttempts: 3,
	}

	if cfg != expected {
		t.Errorf("expected %+v, got %+v", expected, cfg)
	}
}

func TestLoad_S3RequiresBucket(t *testing.T) {
	os.Clearenv()
	t.Setenv("STORAGE_BACKEND", "s3")

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error for missing S3_BUCKET, got nil")
	}

	expectedErr := "S3_BUCKET is required for s3 backend"
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestLoad_InvalidBackend(t *testing.T) {
	os.Clearenv()
	t.Setenv("STORAGE_BACKEND", "floppy")

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error for invalid backend, got nil")
	}

	expectedErr := `unknown STORAGE_BACKEND "floppy" (want local or s3)`
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestLoad_BadMaxAttempts(t *testing.T) {
	os.Clearenv()
	t.Setenv("MAX_ATTEMPTS", "oops")

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error for bad MAX_ATTEMPTS, got nil")
	}

	if !strings.Contains(err.Error(), "MAX_ATTEMPTS=\"oops\"") {
		t.Errorf("expected error to contain the parsing failure, got: %v", err)
	}
}
