package handler

import (
	"errors"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

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
	start := time.Now()
	rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/healthz":
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("ok"))
	case r.Method == http.MethodGet && r.URL.Path == "/readyz":
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("ok"))
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/blob/"):
		h.getBlob(rw, r)
	default:
		http.NotFound(rw, r)
	}

	logLevel := slog.LevelInfo
	if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
		logLevel = slog.LevelDebug
	}
	slog.Log(r.Context(), logLevel, "request",
		"method", r.Method,
		"path", r.URL.Path,
		"status", rw.status,
		"duration_ms", time.Since(start).Milliseconds(),
		"ip", clientIP(r),
		"user_agent", r.UserAgent(),
	)
}

func (h *Handler) getBlob(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/v1/blob/")
	ip := clientIP(r)

	if name == "" {
		http.NotFound(w, r)
		return
	}

	key := extractKey(r)
	if key == "" {
		slog.Warn("access denied: no key provided", "blob", name, "ip", ip) // #nosec G706 -- name is sanitised by store.validateName
		http.Error(w, "missing decryption key", http.StatusUnauthorized)
		return
	}

	ciphertext, err := h.store.Get(name)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			slog.Info("blob not found", "blob", name, "ip", ip) // #nosec G706 -- name is sanitised by store.validateName
			http.NotFound(w, r)
			return
		}
		slog.Error("store read failed", "blob", name, "ip", ip, "err", err) // #nosec G706 -- name is sanitised by store.validateName
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	plaintext, err := crypto.Decrypt(ciphertext, key)
	if err != nil {
		slog.Warn("access denied: decryption failed", "blob", name, "ip", ip) // #nosec G706 -- name is sanitised by store.validateName
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	slog.Info("blob served", "blob", name, "ip", ip, "bytes", len(plaintext)) // #nosec G706 -- name is sanitised by store.validateName

	ct := mime.TypeByExtension(filepath.Ext(name))
	if ct == "" {
		ct = http.DetectContentType(plaintext)
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(plaintext) // #nosec G705 -- plaintext originates from a server-controlled encrypted blob
}

func extractKey(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return r.Header.Get("X-Kryptlet-Key")
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ip, _, _ := strings.Cut(xff, ",")
		return strings.TrimSpace(ip)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type responseWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.status = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}
