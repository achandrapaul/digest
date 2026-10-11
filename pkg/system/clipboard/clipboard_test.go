package clipboard

import (
	"errors"
	"testing"
)

func TestCopySkipsEmptyTextAndPassesTheRest(t *testing.T) {
	previous := writeAll
	t.Cleanup(func() { writeAll = previous })
	var written []string
	writeAll = func(text string) error {
		written = append(written, text)
		return nil
	}
	if err := Copy(""); err != nil {
		t.Fatal(err)
	}
	if err := Copy("hello"); err != nil {
		t.Fatal(err)
	}
	if len(written) != 1 || written[0] != "hello" {
		t.Errorf("written = %q, want only hello", written)
	}
}

func TestCopyReturnsWriteErrors(t *testing.T) {
	previous := writeAll
	t.Cleanup(func() { writeAll = previous })
	failure := errors.New("no clipboard")
	writeAll = func(string) error { return failure }
	if err := Copy("hello"); !errors.Is(err, failure) {
		t.Errorf("err = %v, want %v", err, failure)
	}
}
