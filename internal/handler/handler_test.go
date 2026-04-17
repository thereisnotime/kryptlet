package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"

	"github.com/thereisnotime/kryptlet/internal/handler"
	"github.com/thereisnotime/kryptlet/internal/store"
)

func newTestStore(t *testing.T) (*store.Store, string) {
	t.Helper()
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate identity: %v", err)
	}
	dir := t.TempDir()
	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, identity.Recipient())
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	_, _ = w.Write([]byte(`{"test":true}`))
	_ = w.Close()
	if err := os.WriteFile(filepath.Join(dir, "test.age"), buf.Bytes(), 0o600); err != nil {
		t.Fatalf("write blob: %v", err)
	}
	return store.New(dir), identity.String()
}

func TestHealthz(t *testing.T) {
	h := handler.New(store.New(t.TempDir()))
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("got %d, want 200", rr.Code)
	}
}

func TestReadyz(t *testing.T) {
	h := handler.New(store.New(t.TempDir()))
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("got %d, want 200", rr.Code)
	}
}

func TestGetBlob_BearerToken(t *testing.T) {
	s, key := newTestStore(t)
	h := handler.New(s)
	req := httptest.NewRequest(http.MethodGet, "/v1/blob/test", nil)
	req.Header.Set("Authorization", "Bearer "+key)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("got %d, want 200", rr.Code)
	}
}

func TestGetBlob_XKryptletKeyHeader(t *testing.T) {
	s, key := newTestStore(t)
	h := handler.New(s)
	req := httptest.NewRequest(http.MethodGet, "/v1/blob/test", nil)
	req.Header.Set("X-Kryptlet-Key", key)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("got %d, want 200", rr.Code)
	}
}

func TestGetBlob_MissingKey(t *testing.T) {
	s, _ := newTestStore(t)
	h := handler.New(s)
	req := httptest.NewRequest(http.MethodGet, "/v1/blob/test", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", rr.Code)
	}
}

func TestGetBlob_WrongKey(t *testing.T) {
	s, _ := newTestStore(t)
	h := handler.New(s)
	wrong, _ := age.GenerateX25519Identity()
	req := httptest.NewRequest(http.MethodGet, "/v1/blob/test", nil)
	req.Header.Set("Authorization", "Bearer "+wrong.String())
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", rr.Code)
	}
}

func TestGetBlob_NotFound(t *testing.T) {
	s, key := newTestStore(t)
	h := handler.New(s)
	req := httptest.NewRequest(http.MethodGet, "/v1/blob/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+key)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", rr.Code)
	}
}

func TestGetBlob_EmptyName(t *testing.T) {
	s, key := newTestStore(t)
	h := handler.New(s)
	req := httptest.NewRequest(http.MethodGet, "/v1/blob/", nil)
	req.Header.Set("Authorization", "Bearer "+key)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", rr.Code)
	}
}

func TestUnknownRoute(t *testing.T) {
	h := handler.New(store.New(t.TempDir()))
	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", rr.Code)
	}
}
