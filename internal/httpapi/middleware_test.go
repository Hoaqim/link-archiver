package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func captureLogger() (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	return slog.New(slog.NewJSONHandler(buf, nil)), buf
}

func TestMiddleware_RecoveryAndChainOrder(t *testing.T) {
	panicker := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	logger, logBuf := captureLogger()
	h := withMiddleware(logger, panicker)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	req.Header.Set("X-Request-Id", "fixed-id-for-test")
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rr.Code)
	}

	var found bool
	for _, line := range strings.Split(strings.TrimSpace(logBuf.String()), "\n") {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry["msg"] != "panic recovered" {
			continue
		}
		found = true
		if entry["request_id"] != "fixed-id-for-test" {
			t.Errorf("recovery log request_id = %v, want fixed-id-for-test (chain order regressed?)",
				entry["request_id"])
		}
	}
	if !found {
		t.Errorf("no 'panic recovered' log line\nlogs:\n%s", logBuf.String())
	}
}

func TestMiddleware_RequestIDGenerated(t *testing.T) {
	var seenInHandler string
	h := requestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenInHandler = RequestIDFrom(r.Context())
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	echoed := rr.Header().Get("X-Request-Id")
	if echoed == "" {
		t.Fatal("X-Request-Id not echoed on response")
	}
	if seenInHandler != echoed {
		t.Errorf("handler saw %q, response echoed %q — should match", seenInHandler, echoed)
	}
}

func TestMiddleware_RequestIDPreserved(t *testing.T) {
	const incoming = "trace-from-upstream-7c4f"

	var seenInHandler string
	h := requestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenInHandler = RequestIDFrom(r.Context())
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", incoming)
	h.ServeHTTP(rr, req)

	if seenInHandler != incoming {
		t.Errorf("handler saw %q, want %q", seenInHandler, incoming)
	}
	if echoed := rr.Header().Get("X-Request-Id"); echoed != incoming {
		t.Errorf("response echoed %q, want %q", echoed, incoming)
	}
}

func TestMiddleware_AccessLogCapturesStatus(t *testing.T) {
	cases := []struct {
		name     string
		handler  http.HandlerFunc
		wantCode float64
	}{
		{
			name: "implicit 200 (Write without WriteHeader)",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("ok"))
			},
			wantCode: 200,
		},
		{
			name: "explicit 404",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantCode: 404,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logger, logBuf := captureLogger()
			h := accessLogMiddleware(logger, tc.handler)

			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/jobs", nil))

			var entry map[string]any
			if err := json.Unmarshal(bytes.TrimSpace(logBuf.Bytes()), &entry); err != nil {
				t.Fatalf("log not valid JSON: %v\nbody: %s", err, logBuf.String())
			}
			if entry["status"] != tc.wantCode {
				t.Errorf("status = %v, want %v", entry["status"], tc.wantCode)
			}
		})
	}
}

func TestMiddleware_AccessLogSkipsProbes(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for _, path := range []string{"/health", "/ready"} {
		t.Run(path, func(t *testing.T) {
			logger, logBuf := captureLogger()
			h := accessLogMiddleware(logger, handler)

			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))

			if logBuf.Len() != 0 {
				t.Errorf("expected no log lines for %s, got: %s", path, logBuf.String())
			}
		})
	}
}
