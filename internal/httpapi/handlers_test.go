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

	"github.com/Hoaqim/link-archiver/internal/queue"
	"github.com/Hoaqim/link-archiver/internal/storage"
)

type fakeQueue struct {
	enqueued [][]byte
	err      error
	pingErr  error
}

func (f *fakeQueue) Enqueue(ctx context.Context, payload []byte) error {
	if f.err != nil {
		return f.err
	}
	f.enqueued = append(f.enqueued, payload)
	return nil
}
func (f *fakeQueue) Dequeue(ctx context.Context) (queue.Message, error) { return nil, nil }
func (f *fakeQueue) Ping(ctx context.Context) error                     { return f.pingErr }
func (f *fakeQueue) Close() error                                       { return nil }

type fakeStorage struct {
	data        map[string][]byte
	contentType map[string]string
	getErr      error
	existsErr   error
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{
		data:        map[string][]byte{},
		contentType: map[string]string{},
	}
}

func (f *fakeStorage) Put(ctx context.Context, key string, data []byte, ct string) error {
	f.data[key] = data
	f.contentType[key] = ct
	return nil
}

func (f *fakeStorage) Get(ctx context.Context, key string) ([]byte, string, error) {
	if f.getErr != nil {
		return nil, "", f.getErr
	}
	data, ok := f.data[key]
	if !ok {
		return nil, "", storage.ErrNotFound
	}
	return data, f.contentType[key], nil
}

func (f *fakeStorage) Exists(ctx context.Context, key string) (bool, error) {
	if f.existsErr != nil {
		return false, f.existsErr
	}
	_, ok := f.data[key]
	return ok, nil
}

func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

const testJobID = "11111111-1111-1111-1111-111111111111"

func newGetJobRequest(t *testing.T, id string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/jobs/"+id, nil)
	req.SetPathValue("id", id)
	return req
}

func newJobStatusRequest(t *testing.T, id string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/jobs/"+id+"/status", nil)
	req.SetPathValue("id", id)
	return req
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

func TestReady_OK(t *testing.T) {
	s := &Server{
		Logger:  silentLogger(),
		Queue:   &fakeQueue{},
		Storage: newFakeStorage(),
	}
	rr := httptest.NewRecorder()
	s.Ready(rr, httptest.NewRequest(http.MethodGet, "/ready", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"status":"ready"`) {
		t.Errorf("body = %q, want status:ready", rr.Body.String())
	}
}

func TestReady_QueueDown(t *testing.T) {
	s := &Server{
		Logger:  silentLogger(),
		Queue:   &fakeQueue{pingErr: errors.New("sqs unreachable")},
		Storage: newFakeStorage(),
	}
	rr := httptest.NewRecorder()
	s.Ready(rr, httptest.NewRequest(http.MethodGet, "/ready", nil))

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"reason":"queue"`) {
		t.Errorf("body = %q, want reason:queue", rr.Body.String())
	}
}

func TestReady_StorageDown(t *testing.T) {
	fs := newFakeStorage()
	fs.existsErr = errors.New("s3 unreachable")
	s := &Server{
		Logger:  silentLogger(),
		Queue:   &fakeQueue{},
		Storage: fs,
	}
	rr := httptest.NewRecorder()
	s.Ready(rr, httptest.NewRequest(http.MethodGet, "/ready", nil))

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"reason":"storage"`) {
		t.Errorf("body = %q, want reason:storage", rr.Body.String())
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
	q := &fakeQueue{err: errors.New("sqs send failed")}
	s := &Server{Logger: silentLogger(), Queue: q}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs",
		strings.NewReader(`{"url":"https://example.com"}`))

	s.CreateJob(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rr.Code)
	}
}

func TestGetJob_Hit(t *testing.T) {
	fs := newFakeStorage()
	_ = fs.Put(context.Background(), testJobID+".html", []byte("<h1>hi</h1>"), "text/html")

	s := &Server{Logger: silentLogger(), Storage: fs}
	rr := httptest.NewRecorder()
	s.GetJob(rr, newGetJobRequest(t, testJobID))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if got := rr.Body.String(); got != "<h1>hi</h1>" {
		t.Errorf("body = %q, want '<h1>hi</h1>'", got)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/html" {
		t.Errorf("Content-Type = %q, want 'text/html'", ct)
	}
}

func TestGetJob_Miss(t *testing.T) {
	s := &Server{Logger: silentLogger(), Storage: newFakeStorage()}
	rr := httptest.NewRecorder()
	s.GetJob(rr, newGetJobRequest(t, testJobID))

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestGetJob_InvalidUUID(t *testing.T) {
	s := &Server{Logger: silentLogger(), Storage: newFakeStorage()}
	rr := httptest.NewRecorder()
	s.GetJob(rr, newGetJobRequest(t, "not-a-uuid"))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestGetJob_StorageError(t *testing.T) {
	fs := newFakeStorage()
	fs.getErr = errors.New("disk on fire")
	s := &Server{Logger: silentLogger(), Storage: fs}
	rr := httptest.NewRecorder()
	s.GetJob(rr, newGetJobRequest(t, testJobID))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rr.Code)
	}
}

func TestJobStatus_Done(t *testing.T) {
	fs := newFakeStorage()
	_ = fs.Put(context.Background(), testJobID+".html", []byte("x"), "text/html")

	s := &Server{Logger: silentLogger(), Storage: fs}
	rr := httptest.NewRecorder()
	s.JobStatus(rr, newJobStatusRequest(t, testJobID))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["status"] != "done" {
		t.Errorf("status = %q, want 'done'", resp["status"])
	}
}

func TestJobStatus_Pending(t *testing.T) {
	s := &Server{Logger: silentLogger(), Storage: newFakeStorage()}
	rr := httptest.NewRecorder()
	s.JobStatus(rr, newJobStatusRequest(t, testJobID))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["status"] != "pending" {
		t.Errorf("status = %q, want 'pending'", resp["status"])
	}
}

func TestJobStatus_InvalidUUID(t *testing.T) {
	s := &Server{Logger: silentLogger(), Storage: newFakeStorage()}
	rr := httptest.NewRecorder()
	s.JobStatus(rr, newJobStatusRequest(t, "garbage"))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestJobStatus_StorageError(t *testing.T) {
	fs := newFakeStorage()
	fs.existsErr = errors.New("boom")
	s := &Server{Logger: silentLogger(), Storage: fs}
	rr := httptest.NewRecorder()
	s.JobStatus(rr, newJobStatusRequest(t, testJobID))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rr.Code)
	}
}
