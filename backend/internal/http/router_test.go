package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// failingPinger implements DBPinger and always returns an error.
type failingPinger struct{ err error }

func (f failingPinger) Ping(_ context.Context) error { return f.err }

// successPinger implements DBPinger and always returns nil.
type successPinger struct{}

func (s successPinger) Ping(_ context.Context) error { return nil }

func TestHealthEndpoint(t *testing.T) {
	router := NewRouter(zap.NewNop(), nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %q", body["status"])
	}
}

func TestReadyEndpoint_NoPinger(t *testing.T) {
	// nil pinger → always ready (no DB check).
	router := NewRouter(zap.NewNop(), nil)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if body["status"] != "ready" {
		t.Errorf("expected status ready, got %q", body["status"])
	}
}

func TestReadyEndpoint_PingSuccess(t *testing.T) {
	router := NewRouter(zap.NewNop(), successPinger{})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 when ping succeeds, got %d", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if body["status"] != "ready" {
		t.Errorf("expected status ready, got %q", body["status"])
	}
}

func TestReadyEndpoint_PingFailure(t *testing.T) {
	router := NewRouter(zap.NewNop(), failingPinger{err: errors.New("connection refused")})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when ping fails, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if body["status"] != "not_ready" {
		t.Errorf("expected status not_ready, got %q", body["status"])
	}
	if body["error"] == "" {
		t.Error("expected non-empty error field")
	}
}

func TestRequestIDHeaderSet(t *testing.T) {
	router := NewRouter(zap.NewNop(), nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header to be set")
	}
}
