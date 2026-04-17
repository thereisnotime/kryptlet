package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thereisnotime/kryptlet/internal/handler"
	"github.com/thereisnotime/kryptlet/internal/store"
	"github.com/thereisnotime/kryptlet/internal/version"
)

// Run starts the HTTP server and blocks until SIGINT or SIGTERM is received.
func Run() {
	addr := getEnv("KRYPTLET_ADDR", ":8080")
	blobDir := getEnv("KRYPTLET_BLOB_DIR", "/etc/kryptlet/blobs")

	slog.Info("kryptlet starting",
		"version", version.Version,
		"commit", version.Commit,
		"addr", addr,
		"blobDir", blobDir,
	)

	s := store.New(blobDir)
	h := handler.New(s)

	srv := &http.Server{
		Addr:         addr,
		Handler:      h,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	done := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		slog.Info("shutting down gracefully")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("shutdown error", "err", err)
		}
		close(done)
	}()

	slog.Info("listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
	<-done
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
