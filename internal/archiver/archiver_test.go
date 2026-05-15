package archiver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html>hello</html>"))
	}))
	defer srv.Close()

	a := NewArchiver(5*time.Second, 1<<20)
	res, err := a.Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("status = %d, want 200", res.StatusCode)
	}
	if !strings.Contains(string(res.Body), "hello") {
		t.Errorf("body = %q, want it to contain 'hello'", res.Body)
	}
	if !strings.HasPrefix(res.ContentType, "text/html") {
		t.Errorf("content type = %q, want text/html...", res.ContentType)
	}
}

func TestFetch_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	a := NewArchiver(5*time.Second, 1<<20)
	_, err := a.Fetch(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error = %v, want it to mention 404", err)
	}
}

func TestFetch_BodyTooLarge(t *testing.T) {
	// Server sends 1000 bytes; archiver only allows 100.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(make([]byte, 1000))
	}))
	defer srv.Close()

	a := NewArchiver(5*time.Second, 100)
	_, err := a.Fetch(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected size-limit error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds max size") {
		t.Errorf("error = %v, want size-limit error", err)
	}
}

func TestFetch_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	a := NewArchiver(5*time.Second, 1<<20)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := a.Fetch(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected error when context expires, got nil")
	}
}

func TestFetch_UserAgentSent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	a := NewArchiver(5*time.Second, 1<<20)
	if _, err := a.Fetch(context.Background(), srv.URL); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotUA, "link-archiver") {
		t.Errorf("UA = %q, want it to contain 'link-archiver'", gotUA)
	}
}
