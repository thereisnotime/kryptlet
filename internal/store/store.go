package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNotFound is returned when a blob does not exist in the store.
var ErrNotFound = errors.New("blob not found")

// Store reads age-encrypted blobs from a directory.
// Each blob is a file named <name>.age; the blob name is the filename without the extension.
type Store struct {
	dir string
}

// New creates a Store rooted at dir.
func New(dir string) *Store {
	return &Store{dir: dir}
}

// Get returns the raw encrypted bytes for the named blob, or ErrNotFound if absent.
func (s *Store) Get(name string) ([]byte, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	path := filepath.Join(s.dir, name+".age")
	data, err := os.ReadFile(path) // #nosec G304 — path is sanitised by validateName
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("reading blob: %w", err)
	}
	return data, nil
}

// List returns the names of all blobs in the store (filenames without the .age extension).
func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("reading blob dir: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".age") {
			names = append(names, strings.TrimSuffix(e.Name(), ".age"))
		}
	}
	return names, nil
}

func validateName(name string) error {
	if name == "" || strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return fmt.Errorf("invalid blob name: %q", name)
	}
	return nil
}
