package handler

import (
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/thereisnotime/kryptlet/internal/crypto"
	"github.com/thereisnotime/kryptlet/internal/store"
)

// Handler serves age-encrypted blobs over HTTP.
type Handler struct {
	store *store.Store
}

// New creates a Handler backed by s.
func New(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/healthz":
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	case r.Method == http.MethodGet && r.URL.Path == "/readyz":
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/blob/"):
		h.getBlob(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) getBlob(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/v1/blob/")
	if name == "" {
		http.NotFound(w, r)
		return
	}

	key := extractKey(r)
	if key == "" {
		http.Error(w, "missing decryption key", http.StatusUnauthorized)
		return
	}

	ciphertext, err := h.store.Get(name)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		slog.Error("store read failed", "blob", name, "err", err) // #nosec G706 -- name is sanitised by store.validateName
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	plaintext, err := crypto.Decrypt(ciphertext, key)
	if err != nil {
		slog.Warn("decryption failed", "blob", name) // #nosec G706 -- name is sanitised by store.validateName
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ct := mime.TypeByExtension(filepath.Ext(name))
	if ct == "" {
		ct = http.DetectContentType(plaintext)
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(plaintext) // #nosec G705 -- plaintext originates from a server-controlled encrypted blob, not user input
}

func extractKey(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return r.Header.Get("X-Kryptlet-Key")
}
