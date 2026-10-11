package system

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadTailKeepsWholeLinesFromTheEnd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.log")
	if err := os.WriteFile(path, []byte("first line\nsecond line\nthird\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ReadTail(path, 1024); got != "first line\nsecond line\nthird\n" {
		t.Errorf("a small file is read whole, got %q", got)
	}
	if got := ReadTail(path, 15); got != "third\n" {
		t.Errorf("a cut tail starts at the next whole line, got %q", got)
	}
}

func TestReadTailOfMissingEmptyOrDirectoryIsEmpty(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.log")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(dir, "missing"), empty, dir} {
		if got := ReadTail(path, 1024); got != "" {
			t.Errorf("ReadTail(%s) = %q", path, got)
		}
	}
}
