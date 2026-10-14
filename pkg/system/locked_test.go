package system

import (
	"path/filepath"
	"strconv"
	"sync"
	"testing"
)

func TestLockedUpdateRunsWritersOneAtATime(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "state", "counter.lock")
	counterPath := filepath.Join(dir, "counter")
	const writers = 40
	var group sync.WaitGroup
	for range writers {
		group.Go(func() {
			err := LockedUpdate(lockPath, func() error {
				content, _ := Read(counterPath)
				count, _ := strconv.Atoi(string(content))
				return Write(counterPath, []byte(strconv.Itoa(count+1)))
			})
			if err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if content, _ := Read(counterPath); string(content) != strconv.Itoa(writers) {
		t.Errorf("counter = %q, want %d", content, writers)
	}
}

func TestLockedUpdateReturnsTheUpdateError(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "x.lock")
	want := ErrLockedPath
	if err := LockedUpdate(lockPath, func() error { return want }); err != want {
		t.Errorf("err = %v, want %v", err, want)
	}
}

func TestLockFilesStayUnlocked(t *testing.T) {
	root := protectedRoot(t)
	lockPath := filepath.Join(root, ".state", "app.state.lock")
	if err := LockedUpdate(lockPath, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if locked, err := LockTree(root); err != nil || locked != 0 || isLocked(t, lockPath) {
		t.Errorf("lock files should stay unlocked: locked %d, %v", locked, err)
	}
}
