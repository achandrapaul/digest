package jobs

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/config"
	"github.com/achandrapaul/digest/pkg/system"
)

func tempRuntime(t *testing.T) Runtime {
	t.Helper()
	return NewRuntime(t.TempDir())
}

func writeLogsFile(t *testing.T, runtime Runtime, name, content string) string {
	t.Helper()
	path := filepath.Join(runtime.LogsDir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func blockWithADirectory(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func readOnlyLogsDir(t *testing.T, logsDir string) {
	t.Helper()
	if err := os.Chmod(logsDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(logsDir, 0o700) })
}

func waitUntil(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("condition never became true")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func jobConfig(t *testing.T, command string) *config.Config {
	t.Helper()
	cfg := &config.Config{DigestRoot: t.TempDir(), Jobs: []config.JobSpec{{Name: "nightly", Command: command}}}
	if err := os.MkdirAll(cfg.NotesDir(), 0755); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestNewRuntimeCreatesTheLogsDir(t *testing.T) {
	root := t.TempDir()
	runtime := NewRuntime(root)
	if runtime.LogsDir != filepath.Join(root, "logs") {
		t.Fatalf("logs dir = %s", runtime.LogsDir)
	}
	if info, err := os.Stat(runtime.LogsDir); err != nil || !info.IsDir() {
		t.Fatalf("logs dir not created: %v", err)
	}
}

func TestPIDFilesWithBadOrDeadPIDsAreNotRunning(t *testing.T) {
	runtime := tempRuntime(t)
	writeLogsFile(t, runtime, "garbled.pid", "not a pid")
	if runtime.IsRunning("garbled") {
		t.Fatal("a garbled pid file is not running")
	}
	writeLogsFile(t, runtime, "dead.pid", "999999")
	if _, running := runtime.RunningPID("dead"); running {
		t.Fatal("a dead pid is not running")
	}
}

func TestCheckingAJobLeavesItsStalePIDFile(t *testing.T) {
	runtime := tempRuntime(t)
	pidPath := writeLogsFile(t, runtime, "stale-check.pid", "999999")
	if runtime.IsRunning("stale-check") {
		t.Fatal("a dead pid should not count as running")
	}
	if !fileExists(pidPath) {
		t.Error("checking whether a job runs deleted its pid file")
	}
}

func TestStaleDryRunPIDFilesAreRemoved(t *testing.T) {
	runtime := tempRuntime(t)
	pidPath := writeLogsFile(t, runtime, "nightly.dryrun.pid", "999999")
	if runtime.DryRunInFlight("nightly") {
		t.Fatal("a dead dry run is not in flight")
	}
	if fileExists(pidPath) {
		t.Fatal("the stale pid file should be removed")
	}
}

func TestUnreadableDryRunExitFileCountsAsAFailure(t *testing.T) {
	runtime := tempRuntime(t)
	writeLogsFile(t, runtime, "nightly.dryrun.exit", "???")
	writeLogsFile(t, runtime, "nightly.dryrun.log", "partial output\n")
	output, exitCode, finished := runtime.LoadDryRunResult("nightly")
	if !finished || exitCode != 1 || !strings.Contains(output, "partial output") {
		t.Fatalf("output %q exit %d finished %v", output, exitCode, finished)
	}
}

func TestDryRunOutputKeepsOnlyTheTail(t *testing.T) {
	runtime := tempRuntime(t)
	writeLogsFile(t, runtime, "janitor.dryrun.exit", "0")
	writeLogsFile(t, runtime, "janitor.dryrun.log", strings.Repeat("x", LogTailBytes*3)+"\nwould remove 2 clones\n")
	output, _, ok := runtime.LoadDryRunResult("janitor")
	if !ok || len(output) > LogTailBytes || !strings.HasSuffix(output, "would remove 2 clones\n") {
		t.Errorf("ok %v, read %d bytes, want at most %d ending in the summary", ok, len(output), LogTailBytes)
	}
}

func TestReadLogKeepsOnlyTheTail(t *testing.T) {
	runtime := tempRuntime(t)
	writeLogsFile(t, runtime, "janitor.log", strings.Repeat("x", LogTailBytes*2)+"\nlast line\n")
	logText := runtime.ReadLog("janitor")
	if len(logText) > LogTailBytes || !strings.HasSuffix(logText, "last line\n") {
		t.Errorf("read %d bytes, want the last %d", len(logText), LogTailBytes)
	}
}

func TestDryRunWithoutTheDryRunFlagRecordsTheFailure(t *testing.T) {
	runtime := tempRuntime(t)
	err := runtime.Start(nil, config.JobSpec{Name: "janitor", DryRunCommand: "digest janitor"}, true)
	if !errors.Is(err, ErrDryRunFlagMissing) {
		t.Fatalf("err = %v", err)
	}
	output, exitCode, finished := runtime.LoadDryRunResult("janitor")
	if !finished || exitCode != 1 || !strings.Contains(output, "dry-run command does not pass --dry-run") {
		t.Fatalf("output %q exit %d finished %v", output, exitCode, finished)
	}
}

func TestDryRunFailsWhenTheLogCannotOpen(t *testing.T) {
	runtime := tempRuntime(t)
	readOnlyLogsDir(t, runtime.LogsDir)
	if err := runtime.Start(nil, config.JobSpec{Name: "nightly", DryRunCommand: "true"}, true); err == nil {
		t.Fatal("an unwritable logs dir should fail")
	}
}

func TestDryRunFailsWhenTheShellIsMissing(t *testing.T) {
	runtime := tempRuntime(t)
	t.Setenv("PATH", t.TempDir())
	if err := runtime.Start(nil, config.JobSpec{Name: "nightly", DryRunCommand: "true"}, true); err == nil {
		t.Fatal("a missing shell should fail")
	}
	if _, exitCode, finished := runtime.LoadDryRunResult("nightly"); !finished || exitCode != 1 {
		t.Fatal("the start failure should be recorded")
	}
}

func TestDryRunReportsAnUnwritablePIDFile(t *testing.T) {
	runtime := tempRuntime(t)
	blockWithADirectory(t, runtime.DryRunPath("nightly", "pid"))
	err := runtime.Start(nil, config.JobSpec{Name: "nightly", DryRunCommand: "true"}, true)
	if err == nil || !strings.Contains(err.Error(), "pid file could not be written") {
		t.Fatalf("err = %v", err)
	}
	waitUntil(t, func() bool { _, _, finished := runtime.LoadDryRunResult("nightly"); return finished })
}

func TestDryRunRecordsTheCommandsExitCodeAndOutput(t *testing.T) {
	runtime := tempRuntime(t)
	cases := map[string]struct {
		command  string
		exitCode int
	}{
		"dry-exit":   {command: "echo checked; echo warned >&2; exit 3", exitCode: 3},
		"dry-ok":     {command: "echo checked", exitCode: 0},
		"dry-signal": {command: "echo checked; kill -TERM $$", exitCode: 143},
	}
	for jobName, expected := range cases {
		if err := runtime.Start(nil, config.JobSpec{Name: jobName, DryRunCommand: expected.command}, true); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(5 * time.Second)
		output, exitCode, finished := runtime.LoadDryRunResult(jobName)
		for !finished && time.Now().Before(deadline) {
			time.Sleep(20 * time.Millisecond)
			output, exitCode, finished = runtime.LoadDryRunResult(jobName)
		}
		if !finished {
			t.Fatalf("%s never wrote its exit file", jobName)
		}
		for runtime.DryRunInFlight(jobName) && time.Now().Before(deadline) {
			time.Sleep(20 * time.Millisecond)
		}
		if exitCode != expected.exitCode || !strings.Contains(output, "checked") {
			t.Errorf("%s: exit %d output %q, want exit %d with its output", jobName, exitCode, output, expected.exitCode)
		}
		if jobName == "dry-exit" && !strings.Contains(output, "warned") {
			t.Errorf("stderr missing from the dry-run log: %q", output)
		}
	}
}

func TestBuiltinJobsDefaultToTheRunningBinary(t *testing.T) {
	executable := digestExecutable()
	if command, err := ResolveCommand(config.JobSpec{Name: "janitor"}, false); err != nil || command != executable+" janitor" {
		t.Fatalf("command %q err %v", command, err)
	}
	if command, err := ResolveCommand(config.JobSpec{Name: "janitor"}, true); err != nil || command != executable+" janitor --dry-run" {
		t.Fatalf("dry command %q err %v", command, err)
	}
	if _, err := ResolveCommand(config.JobSpec{Name: "custom"}, false); !errors.Is(err, ErrNoCommand) {
		t.Fatalf("err = %v", err)
	}
	if spec := FindSpec(&config.Config{}, "unknown"); spec.Name != "unknown" || spec.Command != "" {
		t.Fatalf("spec = %+v", spec)
	}
}

func TestDigestDryRunCommandsMustSayDryRun(t *testing.T) {
	_, err := ResolveCommand(config.JobSpec{Name: "janitor", DryRunCommand: "digest janitor --root ~"}, true)
	if !errors.Is(err, ErrDryRunFlagMissing) {
		t.Errorf("a digest dry-run command without --dry-run should be refused, err = %v", err)
	}
	if _, err := ResolveCommand(config.JobSpec{Name: "custom", DryRunCommand: "./check.sh"}, true); err != nil {
		t.Errorf("custom dry-run commands run as written: %v", err)
	}
}

func TestConfiguredDigestCommandsUseTheRunningBinary(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Skip(err)
	}
	for _, dryRun := range []bool{true, false} {
		command, err := ResolveCommand(config.JobSpec{Name: "janitor", Command: "digest janitor --root ~", DryRunCommand: "digest janitor --dry-run --root ~"}, dryRun)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(command, system.ShellQuote(executable)+" janitor ") {
			t.Errorf("dry run %v: command = %q, want it to start with this binary", dryRun, command)
		}
	}
	if command, _ := ResolveCommand(config.JobSpec{Name: "custom", Command: "./digest-like.sh run"}, false); command != "./digest-like.sh run" {
		t.Errorf("non-digest command rewritten: %q", command)
	}
}

func TestHasDryRun(t *testing.T) {
	if HasDryRun("") || !HasDryRun("./check.sh") {
		t.Fatal("only a configured dry-run command counts")
	}
}

func TestJobRunWithoutACommandFails(t *testing.T) {
	runtime := tempRuntime(t)
	cfg := &config.Config{DigestRoot: t.TempDir()}
	if err := runtime.Start(cfg, FindSpec(cfg, "custom"), false); !errors.Is(err, ErrNoCommand) {
		t.Fatalf("err = %v", err)
	}
}

func TestJobRunRotatesThePreviousLogAndCleansItsPIDFile(t *testing.T) {
	runtime := tempRuntime(t)
	writeLogsFile(t, runtime, "nightly.log", "previous run\n")
	cfg := jobConfig(t, "echo fresh run")
	if err := runtime.Start(cfg, FindSpec(cfg, "nightly"), false); err != nil {
		t.Fatal(err)
	}
	archives, _ := filepath.Glob(filepath.Join(runtime.LogsDir, "nightly-*.log"))
	if len(archives) != 1 {
		t.Fatalf("archives = %v", archives)
	}
	if archived, _ := os.ReadFile(archives[0]); string(archived) != "previous run\n" {
		t.Fatalf("archived = %q", archived)
	}
	waitUntil(t, func() bool { return !fileExists(runtime.PIDPath("nightly")) })
	waitUntil(t, func() bool { return strings.Contains(runtime.ReadLog("nightly"), "fresh run") })
}

func TestJobRunFailsWhenTheLogCannotOpen(t *testing.T) {
	runtime := tempRuntime(t)
	readOnlyLogsDir(t, runtime.LogsDir)
	cfg := jobConfig(t, "true")
	if err := runtime.Start(cfg, FindSpec(cfg, "nightly"), false); err == nil {
		t.Fatal("an unwritable logs dir should fail")
	}
}

func TestJobRunFailsWhenTheShellIsMissing(t *testing.T) {
	runtime := tempRuntime(t)
	t.Setenv("PATH", t.TempDir())
	cfg := jobConfig(t, "true")
	if err := runtime.Start(cfg, FindSpec(cfg, "nightly"), false); err == nil {
		t.Fatal("a missing shell should fail")
	}
}

func TestJobRunReportsAnUnwritablePIDFile(t *testing.T) {
	runtime := tempRuntime(t)
	blockWithADirectory(t, runtime.PIDPath("nightly"))
	cfg := jobConfig(t, "true")
	err := runtime.Start(cfg, FindSpec(cfg, "nightly"), false)
	if err == nil || !strings.Contains(err.Error(), "abort unavailable") {
		t.Fatalf("err = %v", err)
	}
}

func TestOldJobLogArchivesArePruned(t *testing.T) {
	logsDir := t.TempDir()
	now := time.Now()
	old, recent, other := "janitor-20200101-000000.log", "janitor-20991231-000000.log", "notes.txt"
	for _, name := range []string{old, recent, other, "janitor.log"} {
		if err := os.WriteFile(filepath.Join(logsDir, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	stale := now.AddDate(0, 0, -30)
	for _, name := range []string{old, other} {
		_ = os.Chtimes(filepath.Join(logsDir, name), stale, stale)
	}
	pruneLogArchives(logsDir, "janitor", 14, now)
	for name, kept := range map[string]bool{old: false, recent: true, other: true, "janitor.log": true} {
		if _, err := os.Stat(filepath.Join(logsDir, name)); (err == nil) != kept {
			t.Errorf("%s kept=%v, want %v", name, err == nil, kept)
		}
	}
}

func TestStopDryRunKillsTheGroupAndRemovesThePIDFile(t *testing.T) {
	runtime := tempRuntime(t)
	sleeper := exec.Command("sleep", "30")
	sleeper.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := sleeper.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan struct{})
	go func() {
		_ = sleeper.Wait()
		close(exited)
	}()
	pidPath := writeLogsFile(t, runtime, "nightly.dryrun.pid", strconv.Itoa(sleeper.Process.Pid))
	if err := runtime.StopDryRun("nightly"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-exited:
	case <-time.After(3 * time.Second):
		_ = sleeper.Process.Kill()
		t.Fatal("the dry run should be killed")
	}
	if fileExists(pidPath) {
		t.Error("the dry-run pid file should be removed")
	}
	if err := runtime.StopDryRun("nightly"); err != nil {
		t.Errorf("stopping with no pid file = %v", err)
	}
}

func TestAbortReportsSignalFailuresForANonGroupProcess(t *testing.T) {
	runtime := tempRuntime(t)
	sleeper := exec.Command("sleep", "30")
	if err := sleeper.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = sleeper.Process.Kill()
		_ = sleeper.Wait()
	})
	writeLogsFile(t, runtime, "lonely.pid", strconv.Itoa(sleeper.Process.Pid))
	if err := runtime.Abort("lonely"); err == nil || !strings.Contains(err.Error(), `failed to kill job "lonely"`) {
		t.Fatalf("err = %v", err)
	}
}

func TestLatestJobOutputPicksTheNewerLog(t *testing.T) {
	runtime := tempRuntime(t)
	if output, isDryRun := runtime.LatestOutput("nightly", "dry output"); output != "dry output" || !isDryRun {
		t.Fatalf("without a real log: %q %v", output, isDryRun)
	}
	realLog := writeLogsFile(t, runtime, "nightly.log", "real output\n")
	dryLog := writeLogsFile(t, runtime, "nightly.dryrun.log", "dry output\n")
	older, newer := time.Now().Add(-time.Hour), time.Now()
	if err := os.Chtimes(realLog, older, older); err != nil {
		t.Fatal(err)
	}
	if output, isDryRun := runtime.LatestOutput("nightly", "dry output"); output != "dry output" || !isDryRun {
		t.Fatalf("newer dry run: %q %v", output, isDryRun)
	}
	if err := os.Chtimes(dryLog, older.Add(-time.Hour), older.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(realLog, newer, newer); err != nil {
		t.Fatal(err)
	}
	if output, isDryRun := runtime.LatestOutput("nightly", "dry output"); !strings.Contains(output, "real output") || isDryRun {
		t.Fatalf("newer real run: %q %v", output, isDryRun)
	}
	if !modTime(filepath.Join(t.TempDir(), "missing")).IsZero() {
		t.Fatal("a missing log has no mod time")
	}
}

func TestStampsChangeWhenLogsChange(t *testing.T) {
	runtime := tempRuntime(t)
	emptyDryRun, emptyLog := runtime.DryRunStamp("nightly"), runtime.LogStamp("nightly", false)
	writeLogsFile(t, runtime, "nightly.dryrun.log", "dry\n")
	writeLogsFile(t, runtime, "nightly.log", "real\n")
	if runtime.DryRunStamp("nightly") == emptyDryRun || runtime.LogStamp("nightly", false) == emptyLog {
		t.Fatal("writing logs should change their stamps")
	}
	if runtime.LogStamp("nightly", true) == runtime.LogStamp("nightly", false) {
		t.Fatal("the running flag is part of the log stamp")
	}
}
