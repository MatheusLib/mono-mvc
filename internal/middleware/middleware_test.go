package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"io"
	"testing"
)

func okHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func errorHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

// ── RequestID ─────────────────────────────────────────────────────────────────

func TestRequestID_GeneratesID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	RequestID(http.HandlerFunc(okHandler)).ServeHTTP(w, req)

	if w.Header().Get("X-Request-Id") == "" {
		t.Fatal("expected X-Request-Id header to be set")
	}
}

func TestRequestID_PreservesExistingID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "my-custom-id")
	w := httptest.NewRecorder()

	RequestID(http.HandlerFunc(okHandler)).ServeHTTP(w, req)

	if got := w.Header().Get("X-Request-Id"); got != "my-custom-id" {
		t.Fatalf("expected my-custom-id, got %s", got)
	}
}

func TestGetRequestID_Missing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if id := GetRequestID(req.Context()); id != "" {
		t.Fatalf("expected empty string, got %q", id)
	}
}

func TestGetRequestID_Present(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	var capturedID string
	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedID = GetRequestID(r.Context())
	})

	RequestID(inner).ServeHTTP(w, req)

	if capturedID == "" {
		t.Fatal("expected request ID in context")
	}
}

// ── Logging ───────────────────────────────────────────────────────────────────

func TestLogging_Success(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	Logging(logger)(http.HandlerFunc(okHandler)).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestLogging_Error(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	Logging(logger)(http.HandlerFunc(errorHandler)).ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── Tracing ───────────────────────────────────────────────────────────────────

func TestTracing_Success(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	Tracing(http.HandlerFunc(okHandler)).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestTracing_ServerError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	Tracing(http.HandlerFunc(errorHandler)).ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── statusWriter ──────────────────────────────────────────────────────────────

func TestStatusWriter_DefaultStatus(t *testing.T) {
	w := httptest.NewRecorder()
	sw := newStatusWriter(w)
	_, _ = sw.Write([]byte("hello"))

	if sw.status != http.StatusOK {
		t.Fatalf("expected default status 200, got %d", sw.status)
	}
	if sw.bytes != 5 {
		t.Fatalf("expected 5 bytes written, got %d", sw.bytes)
	}
}

func TestStatusWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	sw := newStatusWriter(w)
	sw.WriteHeader(http.StatusCreated)

	if sw.status != http.StatusCreated {
		t.Fatalf("expected 201, got %d", sw.status)
	}
}

func TestEnsureStatusWriter_AlreadyWrapped(t *testing.T) {
	w := httptest.NewRecorder()
	sw := newStatusWriter(w)

	sw2, rw := ensureStatusWriter(sw)
	if sw2 != sw {
		t.Fatal("expected same statusWriter to be returned")
	}
	if rw != sw {
		t.Fatal("expected same writer to be returned")
	}
}
