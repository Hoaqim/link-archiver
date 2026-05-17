package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type mockS3 struct {
	putCalled int
	putInput  *s3.PutObjectInput
	getOut    *s3.GetObjectOutput
	getErr    error
	headErr   error
}

func (m *mockS3) PutObject(ctx context.Context, in *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	m.putCalled++
	m.putInput = in
	return &s3.PutObjectOutput{}, nil
}
func (m *mockS3) GetObject(ctx context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.getOut, nil
}
func (m *mockS3) HeadObject(ctx context.Context, _ *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if m.headErr != nil {
		return nil, m.headErr
	}
	return &s3.HeadObjectOutput{}, nil
}

const testBucket = "test-bucket"

func newTestS3(client s3API) *S3 {
	return &S3{client: client, bucket: testBucket}
}

func TestS3_PutSetsContentType(t *testing.T) {
	mock := &mockS3{}
	s := newTestS3(mock)

	if err := s.Put(context.Background(), "k", []byte("<h1>x</h1>"), "text/html"); err != nil {
		t.Fatal(err)
	}
	if mock.putCalled != 1 {
		t.Fatalf("putCalled = %d, want 1", mock.putCalled)
	}
	if got := aws.ToString(mock.putInput.ContentType); got != "text/html" {
		t.Errorf("ContentType = %q, want text/html", got)
	}
}

func TestS3_GetMissingReturnsErrNotFound(t *testing.T) {
	mock := &mockS3{getErr: &s3types.NoSuchKey{}}
	s := newTestS3(mock)

	_, _, err := s.Get(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

// Get must surface body bytes and content-type from the SDK response.
func TestS3_GetReturnsBodyAndContentType(t *testing.T) {
	mock := &mockS3{
		getOut: &s3.GetObjectOutput{
			Body:        io.NopCloser(bytes.NewReader([]byte("hello"))),
			ContentType: aws.String("text/plain"),
		},
	}
	s := newTestS3(mock)

	data, ct, err := s.Get(context.Background(), "k")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Errorf("data = %q, want hello", data)
	}
	if ct != "text/plain" {
		t.Errorf("contentType = %q, want text/plain", ct)
	}
}

// Exists must return (false, nil) for a missing key — NOT an error. The
// /ready probe uses Exists on a sentinel key to verify reachability; an
// error response would flag the bucket as unhealthy on every probe.
func TestS3_ExistsHandlesMissingCleanly(t *testing.T) {
	mock := &mockS3{headErr: &s3types.NotFound{}}
	s := newTestS3(mock)

	ok, err := s.Exists(context.Background(), "missing")
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if ok {
		t.Error("ok = true, want false")
	}
}
