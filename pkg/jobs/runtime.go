package jobs

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/achandrapaul/digest/pkg/review"
	"github.com/achandrapaul/digest/pkg/system"
	"github.com/achandrapaul/digest/pkg/system/process"
)

const (
	LogTailBytes = 64 * 1024
	LogPeekBytes = 16 * 1024
)

type Runtime struct {
	LogsDir string
}

var createdLogsDirs sync.Map

func NewRuntime(root string) Runtime {
	dir := filepath.Join(root, "logs")
	if _, created := createdLogsDirs.Load(dir); !created {
		if system.MkdirAll(dir) == nil {
			createdLogsDirs.Store(dir, true)
		}
	}
	return Runtime{LogsDir: dir}
}

func (r Runtime) LogPath(name string) string {
	return filepath.Join(r.LogsDir, name+".log")
}

func (r Runtime) PIDPath(name string) string {
	return filepath.Join(r.LogsDir, name+".pid")
}

func (r Runtime) DryRunPath(name, suffix string) string {
	return filepath.Join(r.LogsDir, fmt.Sprintf("%s.dryrun.%s", name, suffix))
}

func (r Runtime) RunningPID(name string) (int, bool) {
	pid, ok := process.ReadInt(r.PIDPath(name))
	if !ok || !process.PIDAlive(pid) {
		return 0, false
	}
	return pid, true
}

func (r Runtime) IsRunning(name string) bool {
	_, running := r.RunningPID(name)
	return running
}

func (r Runtime) DryRunInFlight(name string) bool {
	pidFile := r.DryRunPath(name, "pid")
	if !system.Exists(pidFile) {
		return false
	}
	if pid, ok := process.ReadInt(pidFile); ok && process.PIDAlive(pid) {
		return true
	}
	_ = system.Remove(pidFile)
	return false
}

func (r Runtime) LoadDryRunResult(name string) (string, int, bool) {
	exitData, err := system.Read(r.DryRunPath(name, "exit"))
	if err != nil {
		return "", 0, false
	}
	exitCode, err := strconv.Atoi(strings.TrimSpace(string(exitData)))
	if err != nil {
		exitCode = 1
	}
	return readTail(r.DryRunPath(name, "log"), LogTailBytes), exitCode, true
}

func (r Runtime) DryRunStamp(name string) string {
	var stamp strings.Builder
	for _, suffix := range []string{"exit", "log"} {
		if info, err := system.Stat(r.DryRunPath(name, suffix)); err == nil {
			fmt.Fprintf(&stamp, "%s:%d:%d|", suffix, info.Size(), info.ModTime().UnixNano())
		}
	}
	return stamp.String()
}

func (r Runtime) LogStamp(name string, running bool) string {
	stamp := strconv.FormatBool(running)
	for _, path := range []string{r.LogPath(name), r.DryRunPath(name, "log")} {
		if info, err := system.Stat(path); err == nil {
			stamp += fmt.Sprintf("|%d:%d", info.Size(), info.ModTime().UnixNano())
		}
	}
	return stamp
}

func readTail(path string, maxBytes int64) string {
	return review.PlainText(system.ReadTail(path, maxBytes))
}

func (r Runtime) ReadLog(name string) string {
	return readTail(r.LogPath(name), LogTailBytes)
}

func modTime(path string) time.Time {
	info, err := system.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func (r Runtime) LatestOutput(name, dryRunOutput string) (string, bool) {
	realLog := r.ReadLog(name)
	if r.IsRunning(name) || strings.TrimSpace(dryRunOutput) == "" {
		return realLog, false
	}
	if realLog == "" {
		return dryRunOutput, true
	}
	if modTime(r.DryRunPath(name, "log")).After(modTime(r.LogPath(name))) {
		return dryRunOutput, true
	}
	return realLog, false
}
