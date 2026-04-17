package store_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/thereisnotime/kryptlet/internal/store"
)

func TestGet(t *testing.T) {
	dir := t.TempDir()
	want := []byte("encrypted-content")
	if err := os.WriteFile(filepath.Join(dir, "myblob.age"), want, 0o600); err != nil {
		t.Fatal(err)
	}
	s := store.New(dir)
	got, err := s.Get("myblob")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGet_NotFound(t *testing.T) {
	s := store.New(t.TempDir())
	_, err := s.Get("nonexistent")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGet_InvalidName(t *testing.T) {
	s := store.New(t.TempDir())
	for _, name := range []string{"../etc/passwd", "foo/bar", "", "..secret", "..", "a/b", "a\\b"} {
		if _, err := s.Get(name); err == nil {
			t.Errorf("expected error for name %q, got nil", name)
		}
	}
}

func TestGet_DotInName(t *testing.T) {
	dir := t.TempDir()
	want := []byte("content")
	if err := os.WriteFile(filepath.Join(dir, "test.json.age"), want, 0o600); err != nil {
		t.Fatal(err)
	}
	s := store.New(dir)
	got, err := s.Get("test.json")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"a.age", "b.age", "other.txt", "skip.yaml"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	s := store.New(dir)
	names, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 2 {
		t.Errorf("got %d blobs, want 2: %v", len(names), names)
	}
}

func TestList_EmptyDir(t *testing.T) {
	s := store.New(t.TempDir())
	names, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected empty list, got %v", names)
	}
}
