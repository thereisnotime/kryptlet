package version_test

import (
	"strings"
	"testing"

	"github.com/thereisnotime/kryptlet/internal/version"
)

func TestString(t *testing.T) {
	s := version.String()
	if !strings.HasPrefix(s, "kryptlet ") {
		t.Errorf("version string %q does not start with 'kryptlet '", s)
	}
}
