package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeQueue implements queue.Queue for tests. We only care about Enqueue
// for these handler tests; Dequeue/Close just satisfy the interface.
type fakeQueue struct {
	enqueued [][]byte
	err      error // if non-nil, Enqueue returns this
}

func (f *fakeQueue) Enqueue(ctx context.Context, payload []byte) error {
	if f.err != nil {
		return f.err
	}
	f.enqueued = append(f.enqueued, payload)
	return nil
}
func (f *fakeQueue) Dequeue(ctx context.Context) ([]byte, error) { return nil, nil }
func (f *fakeQueue) Close() error                                { return nil }

// silentLogger discards log output so test runs aren't noisy.
func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHealth(t *testing.T) {
	s := &Server{Logger: silentLogger()}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	s.Health(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"status":"ok"`) {
		t.Errorf("body = %q, want it to contain status:ok", rr.Body.String())
	}
}

func TestCreateJob_Success(t *testing.T) {
	q := &fakeQueue{}
	s := &Server{Logger: silentLogger(), Queue: q}

	body := bytes.NewBufferString(`{"url":"https://example.com"}`)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs", body)

	s.CreateJob(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("status = %d, want 202", rr.Code)
	}
	if len(q.enqueued) != 1 {
		t.Fatalf("enqueued %d jobs, want 1", len(q.enqueued))
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["id"] == "" {
		t.Error("response missing id")
	}
}

func TestCreateJob_InvalidJSON(t *testing.T) {
	s := &Server{Logger: silentLogger(), Queue: &fakeQueue{}}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader("not json"))

	s.CreateJob(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestCreateJob_MissingURL(t *testing.T) {
	s := &Server{Logger: silentLogger(), Queue: &fakeQueue{}}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"url":""}`))

	s.CreateJob(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestCreateJob_QueueError(t *testing.T) {
	q := &fakeQueue{err: errors.New("redis down")}
	s := &Server{Logger: silentLogger(), Queue: q}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"url":"https://example.com"}`))

	s.CreateJob(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rr.Code)
	}
}
