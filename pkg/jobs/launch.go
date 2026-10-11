package jobs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/achandrapaul/digest/pkg/config"
	"github.com/achandrapaul/digest/pkg/system"
	"github.com/achandrapaul/digest/pkg/system/process"
)

const logArchiveFormat = "20060102-150405"

var abortGracePeriod = 3 * time.Second

func pruneLogArchives(logsDir, name string, retentionDays int, now time.Time) {
	archives, _ := system.Glob(filepath.Join(logsDir, name+"-*.log"))
	cutoff := now.AddDate(0, 0, -retentionDays)
	for _, archive := range archives {
		stamp := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(archive), name+"-"), ".log")
		if _, err := time.Parse(logArchiveFormat, stamp); err != nil {
			continue
		}
		if info, err := system.Stat(archive); err == nil && info.ModTime().Before(cutoff) {
			_ = system.Remove(archive)
		}
	}
}

type launch struct {
	runtime Runtime
	cfg     *config.Config
	spec    config.JobSpec
	dryRun  bool
}

type launchFiles struct {
	log, pid, exit string
}

func (job launch) files() launchFiles {
	name := job.spec.Name
	if job.dryRun {
		return launchFiles{log: job.runtime.DryRunPath(name, "log"), pid: job.runtime.DryRunPath(name, "pid"), exit: job.runtime.DryRunPath(name, "exit")}
	}
	return launchFiles{log: job.runtime.LogPath(name), pid: job.runtime.PIDPath(name)}
}

func (job launch) failed(err error) error {
	if job.dryRun {
		_ = system.Write(job.runtime.DryRunPath(job.spec.Name, "log"), []byte(err.Error()+"\n"))
		_ = system.Write(job.runtime.DryRunPath(job.spec.Name, "exit"), []byte("1"))
	}
	return err
}

func (job launch) prepareLog(files launchFiles) {
	if job.dryRun {
		_ = system.Remove(files.exit)
		return
	}
	if system.Exists(files.log) {
		archivedLog := filepath.Join(filepath.Dir(files.log), fmt.Sprintf("%s-%s.log", job.spec.Name, time.Now().Format(logArchiveFormat)))
		_ = system.Rename(files.log, archivedLog)
	}
	pruneLogArchives(filepath.Dir(files.log), job.spec.Name, job.cfg.Retention(), time.Now())
}

func (job launch) command(commandStr string, files launchFiles) *exec.Cmd {
	cmd := exec.Command("sh", "-c", commandStr)
	if job.dryRun {
		cmd = exec.Command("sh", "-c", `sh -c "$1"; echo $? > "$2"`, "digest-dry-run", commandStr, files.exit)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Env = append(os.Environ(), job.spec.OptionsEnv()...)
	if home, err := os.UserHomeDir(); err == nil {
		cmd.Dir = home
	}
	if !job.dryRun && job.cfg != nil {
		cmd.Dir = job.cfg.NotesDir()
	}
	return cmd
}

func (job launch) pidWriteError(err error) error {
	if job.dryRun {
		return fmt.Errorf("dry run for %q started but its pid file could not be written: %w", job.spec.Name, err)
	}
	return fmt.Errorf("job %q started but its pid file could not be written (abort unavailable): %w", job.spec.Name, err)
}

func (r Runtime) Start(cfg *config.Config, spec config.JobSpec, dryRun bool) error {
	job := launch{runtime: r, cfg: cfg, spec: spec, dryRun: dryRun}
	commandStr, err := ResolveCommand(spec, dryRun)
	if err != nil {
		return job.failed(err)
	}
	files := job.files()
	job.prepareLog(files)
	logFile, err := system.OpenLog(files.log)
	if err != nil {
		return job.failed(err)
	}
	defer logFile.Close()

	cmd := job.command(commandStr, files)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		return job.failed(err)
	}

	ownPid := strconv.Itoa(cmd.Process.Pid)
	pidWriteErr := system.Write(files.pid, []byte(ownPid))

	go func() {
		_ = cmd.Wait()
		if data, err := system.Read(files.pid); err == nil && strings.TrimSpace(string(data)) == ownPid {
			_ = system.Remove(files.pid)
		}
	}()

	if pidWriteErr != nil {
		return job.pidWriteError(pidWriteErr)
	}
	return nil
}

func (r Runtime) StopDryRun(name string) error {
	pidFile := r.DryRunPath(name, "pid")
	if !system.Exists(pidFile) {
		return nil
	}
	if pid, ok := process.ReadInt(pidFile); ok && process.PIDAlive(pid) {
		if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			return fmt.Errorf("failed to stop the dry run of %q: %w", name, err)
		}
	}
	if err := system.Remove(pidFile); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (r Runtime) Abort(name string) error {
	pidFile := r.PIDPath(name)
	data, err := system.Read(pidFile)
	if err != nil {
		return fmt.Errorf("failed to read pid file for %q: %w", name, err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return fmt.Errorf("invalid pid file for %q: %w", name, err)
	}

	var abortErr error
	if process.PIDAlive(pid) {
		if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil {
			abortErr = fmt.Errorf("failed to terminate job %q: %w", name, err)
		}
		deadline := time.Now().Add(abortGracePeriod)
		for process.PIDAlive(pid) && time.Now().Before(deadline) {
			time.Sleep(100 * time.Millisecond)
		}
		if process.PIDAlive(pid) {
			if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
				abortErr = fmt.Errorf("failed to kill job %q: %w", name, err)
			}
		}
	}

	if activeLog := r.LogPath(name); system.Exists(activeLog) {
		_ = system.Append(activeLog, []byte("\n[JOB ABORTED BY USER]\n"))
	}

	if current, err := system.Read(pidFile); err == nil && strings.TrimSpace(string(current)) == strconv.Itoa(pid) {
		_ = system.Remove(pidFile)
	}
	return abortErr
}
