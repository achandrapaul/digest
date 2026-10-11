package process

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

const goneProcessID = 1 << 22

func TestPIDAliveForThisProcessOnly(t *testing.T) {
	if !PIDAlive(os.Getpid()) {
		t.Error("this process should be alive")
	}
	for _, pid := range []int{0, -4, goneProcessID} {
		if PIDAlive(pid) {
			t.Errorf("pid %d should not be alive", pid)
		}
	}
}

func TestReadIntParsesTrimmedNumbers(t *testing.T) {
	dir := t.TempDir()
	numberPath := filepath.Join(dir, "run.pid")
	if err := os.WriteFile(numberPath, []byte(" 42\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if value, ok := ReadInt(numberPath); !ok || value != 42 {
		t.Errorf("ReadInt = %d, %v; want 42, true", value, ok)
	}
	garbagePath := filepath.Join(dir, "garbage")
	if err := os.WriteFile(garbagePath, []byte("nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := ReadInt(garbagePath); ok {
		t.Error("garbage should not parse")
	}
	if _, ok := ReadInt(filepath.Join(dir, "missing")); ok {
		t.Error("a missing file should not parse")
	}
}

func TestClearExitedRemovesOnlyDeadPIDs(t *testing.T) {
	dir := t.TempDir()
	deadPath := filepath.Join(dir, "dead.pid")
	livePath := filepath.Join(dir, "live.pid")
	if err := os.WriteFile(deadPath, []byte(strconv.Itoa(goneProcessID)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(livePath, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}
	ClearExited(deadPath)
	ClearExited(livePath)
	if _, err := os.Stat(deadPath); !os.IsNotExist(err) {
		t.Error("a dead pid file should be removed")
	}
	if _, err := os.Stat(livePath); err != nil {
		t.Error("a live pid file should be kept")
	}
}

func TestTerminateGroupKillsWhatIgnoresTerm(t *testing.T) {
	child := exec.Command("sh", "-c", `trap "" TERM; sleep 30`)
	child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan struct{})
	go func() {
		_ = child.Wait()
		close(exited)
	}()
	time.Sleep(100 * time.Millisecond)
	if err := TerminateGroup(child.Process.Pid, 200*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	select {
	case <-exited:
	case <-time.After(3 * time.Second):
		_ = syscall.Kill(-child.Process.Pid, syscall.SIGKILL)
		t.Fatal("a group ignoring TERM should be killed after the grace period")
	}
}

func TestSpawnDetachedWritesPIDLogAndRunsOnExit(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "run.log")
	pidPath := filepath.Join(dir, "run.pid")
	exitCodes := make(chan int, 1)
	err := SpawnDetached(DetachSpec{
		Label:   "test run",
		Args:    []string{"sh", "-c", "echo hello; exit 3"},
		Dir:     dir,
		LogPath: logPath,
		PIDPath: pidPath,
		OnExit:  func(cmd *exec.Cmd) { exitCodes <- cmd.ProcessState.ExitCode() },
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case code := <-exitCodes:
		if code != 3 {
			t.Errorf("exit code = %d, want 3", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("OnExit never ran")
	}
	logged, err := os.ReadFile(logPath)
	if err != nil || string(logged) != "hello\n" {
		t.Errorf("log = %q, %v; want hello", logged, err)
	}
	if _, ok := ReadInt(pidPath); !ok {
		t.Error("the pid file should hold the child pid")
	}
}

func TestSpawnDetachedReportsStartFailure(t *testing.T) {
	dir := t.TempDir()
	err := SpawnDetached(DetachSpec{
		Label:   "test run",
		Args:    []string{filepath.Join(dir, "missing-binary")},
		Dir:     dir,
		LogPath: filepath.Join(dir, "run.log"),
		PIDPath: filepath.Join(dir, "run.pid"),
	})
	if err == nil {
		t.Fatal("starting a missing binary should fail")
	}
}
