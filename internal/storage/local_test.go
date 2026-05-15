package storage

import (
	"context"
	"strings"
	"testing"
)

func TestLocal_PutGet(t *testing.T) {
	l, err := NewLocal(t.TempDir()) // t.TempDir auto-cleans after the test
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if err := l.Put(ctx, "abc.html", []byte("<h1>hi</h1>"), "text/html"); err != nil {
		t.Fatal(err)
	}

	data, ct, err := l.Get(ctx, "abc.html")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "<h1>hi</h1>" {
		t.Errorf("data = %q, want '<h1>hi</h1>'", data)
	}
	if ct != "text/html" {
		t.Errorf("ct = %q, want 'text/html'", ct)
	}
}

func TestLocal_Exists(t *testing.T) {
	l, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	ok, err := l.Exists(ctx, "missing")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("Exists returned true for a missing key")
	}

	if err := l.Put(ctx, "here", []byte("x"), "text/plain"); err != nil {
		t.Fatal(err)
	}
	ok, err = l.Exists(ctx, "here")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("Exists returned false after Put")
	}
}

func TestLocal_GetMissing(t *testing.T) {
	l, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = l.Get(context.Background(), "nope")
	if err == nil {
		t.Fatal("expected error for missing key, got nil")
	}
	if !strings.Contains(err.Error(), "storage: not found") {
		t.Errorf("error = %v, want it to mention 'storage: not found'", err)
	}
}
