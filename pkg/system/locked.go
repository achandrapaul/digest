package system

import (
	"errors"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/sys/unix"

	"github.com/achandrapaul/digest/pkg/paths"
)

var updateLocks sync.Map

func LockedUpdate(lockPath string, update func() error) error {
	inProcess, _ := updateLocks.LoadOrStore(filepath.Clean(lockPath), &sync.Mutex{})
	mutex := inProcess.(*sync.Mutex)
	mutex.Lock()
	defer mutex.Unlock()
	if err := os.MkdirAll(filepath.Dir(lockPath), paths.PrivateDirMode); err != nil {
		return err
	}
	lockFile, err := os.OpenFile(lockPath, os.O_RDONLY|os.O_CREATE, paths.PrivateFileMode)
	if err != nil {
		return err
	}
	defer lockFile.Close()
	for {
		err = unix.Flock(int(lockFile.Fd()), unix.LOCK_EX)
		if !errors.Is(err, unix.EINTR) {
			break
		}
	}
	if err != nil {
		return &os.PathError{Op: "flock", Path: lockPath, Err: err}
	}
	defer unix.Flock(int(lockFile.Fd()), unix.LOCK_UN)
	return update()
}
