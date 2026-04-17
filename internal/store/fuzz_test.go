package store_test

import (
	"testing"

	"github.com/thereisnotime/kryptlet/internal/store"
)

func FuzzGet(f *testing.F) {
	// Seed with path traversal attempts and edge cases.
	f.Add("config")
	f.Add("../etc/passwd")
	f.Add("../../secret")
	f.Add("foo/bar")
	f.Add("foo\\bar")
	f.Add("")
	f.Add(".")
	f.Add("..")
	f.Add("a.age")
	f.Add(string([]byte{0x00}))

	f.Fuzz(func(t *testing.T, name string) {
		s := store.New(t.TempDir())
		// Must never panic — only return an error or ErrNotFound.
		_, _ = s.Get(name)
	})
}
