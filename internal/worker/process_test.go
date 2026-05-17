package worker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Hoaqim/link-archiver/internal/archiver"
	"github.com/Hoaqim/link-archiver/internal/queue"
)

type fakeMessage struct {
	payload []byte
	acked   bool
	nacked  bool
}

func (m *fakeMessage) Payload() []byte                { return m.payload }
func (m *fakeMessage) ReceiveCount() int              { return 1 }
func (m *fakeMessage) Ack(ctx context.Context) error  { m.acked = true; return nil }
func (m *fakeMessage) Nack(ctx context.Context) error { m.nacked = true; return nil }

type fakeStorage struct{ putErr error }

func (f *fakeStorage) Put(ctx context.Context, k string, d []byte, ct string) error {
	return f.putErr
}
func (f *fakeStorage) Get(ctx context.Context, k string) ([]byte, string, error) {
	return nil, "", errors.New("not used in worker tests")
}
func (f *fakeStorage) Exists(ctx context.Context, k string) (bool, error) {
	return false, nil
}

func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestProcessor(store *fakeStorage) *Processor {
	return &Processor{
		Logger:   silentLogger(),
		Archiver: archiver.NewArchiver(2*time.Second, 1<<20),
		Storage:  store,
	}
}

func mustMarshal(t *testing.T, j queue.Job) []byte {
	t.Helper()
	b, err := json.Marshal(j)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

const testJobID = "11111111-1111-1111-1111-111111111111"

func TestProcess_SuccessAcks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<h1>ok</h1>"))
	}))
	defer srv.Close()

	p := newTestProcessor(&fakeStorage{})
	msg := &fakeMessage{payload: mustMarshal(t, queue.Job{ID: testJobID, URL: srv.URL})}

	if err := p.Process(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	if !msg.acked {
		t.Error("expected Ack, got none")
	}
	if msg.nacked {
		t.Error("expected no Nack, got one")
	}
}

func TestProcess_FetchFailureNacks(t *testing.T) {
	p := newTestProcessor(&fakeStorage{})
	msg := &fakeMessage{payload: mustMarshal(t, queue.Job{ID: testJobID, URL: "http://127.0.0.1:1"})}

	if err := p.Process(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	if msg.acked {
		t.Error("expected no Ack on fetch failure, got one")
	}
	if !msg.nacked {
		t.Error("expected Nack, got none")
	}
}

func TestProcess_UnmarshalFailureNacks(t *testing.T) {
	p := newTestProcessor(&fakeStorage{})
	msg := &fakeMessage{payload: []byte("not json")}

	if err := p.Process(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	if msg.acked {
		t.Error("expected no Ack on poisoned payload, got one")
	}
	if !msg.nacked {
		t.Error("expected Nack, got none")
	}
}

func TestProcess_ContextCancelledDoesNeither(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	p := newTestProcessor(&fakeStorage{})
	msg := &fakeMessage{payload: mustMarshal(t, queue.Job{ID: testJobID, URL: srv.URL})}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := p.Process(ctx, msg)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if msg.acked || msg.nacked {
		t.Errorf("expected no Ack/Nack during shutdown, acked=%v nacked=%v",
			msg.acked, msg.nacked)
	}
}
