package process

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/achandrapaul/digest/pkg/system"
)

func PIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	running, err := os.FindProcess(pid)
	return err == nil && running.Signal(syscall.Signal(0)) == nil
}

func ReadInt(path string) (int, bool) {
	data, err := system.Read(path)
	if err != nil {
		return 0, false
	}
	value, err := strconv.Atoi(strings.TrimSpace(string(data)))
	return value, err == nil
}

func ClearExited(pidPath string) {
	if pid, ok := ReadInt(pidPath); ok && !PIDAlive(pid) {
		_ = system.Remove(pidPath)
	}
}

func groupTarget(pid int) int {
	if groupID, err := syscall.Getpgid(pid); err == nil && groupID != syscall.Getpgrp() {
		return -groupID
	}
	return pid
}

func TerminateGroup(pid int, grace time.Duration) error {
	target := groupTarget(pid)
	if err := syscall.Kill(target, syscall.SIGTERM); err != nil {
		return err
	}
	deadline := time.Now().Add(grace)
	go func() {
		for time.Now().Before(deadline) {
			if syscall.Kill(target, 0) != nil {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		_ = syscall.Kill(target, syscall.SIGKILL)
	}()
	return nil
}
