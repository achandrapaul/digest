package system

import (
	"context"
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestRunShellFeedsStdinAndSplitsOutput(t *testing.T) {
	dir := t.TempDir()
	stdout, stderr, err := RunShell(context.Background(), dir, `cat; echo oops >&2; pwd`, "hello\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stdout, "hello\n") || !strings.Contains(stdout, dir[strings.LastIndex(dir, "/"):]) {
		t.Errorf("stdout = %q, want stdin echoed then the dir", stdout)
	}
	if strings.TrimSpace(stderr) != "oops" {
		t.Errorf("stderr = %q, want oops", stderr)
	}
}

func TestRunShellReturnsExitErrors(t *testing.T) {
	_, _, err := RunShell(context.Background(), "", "exit 4", "")
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 4 {
		t.Fatalf("err = %v, want exit status 4", err)
	}
}

func TestShellCommandKillsTheWholeGroupOnCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	started := time.Now()
	_ = ShellCommand(ctx, "sleep 30 & sleep 30").Run()
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("cancel took %s, the process group was not killed", elapsed)
	}
}

func TestGitCommandUsesTheSafeEnv(t *testing.T) {
	cmd := GitCommand(context.Background(), "/tmp", "status")
	if cmd.Dir != "/tmp" || !slices.Equal(cmd.Args, []string{"git", "status"}) {
		t.Fatalf("dir %q args %v", cmd.Dir, cmd.Args)
	}
	for _, want := range append([]string{"GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0"}, GitSafeEnv()...) {
		if !slices.Contains(cmd.Env, want) {
			t.Errorf("env is missing %s", want)
		}
	}
	if cmd.WaitDelay != CommandWaitDelay {
		t.Errorf("wait delay = %s", cmd.WaitDelay)
	}
}

func TestGHCommandSetsWaitDelay(t *testing.T) {
	cmd := GHCommand(context.Background(), "auth", "status")
	if !slices.Equal(cmd.Args, []string{"gh", "auth", "status"}) || cmd.WaitDelay != CommandWaitDelay {
		t.Fatalf("args %v wait %s", cmd.Args, cmd.WaitDelay)
	}
}

func TestShellQuoteSurvivesTheShell(t *testing.T) {
	for _, text := range []string{"plain", "it's", "a b", `$HOME "x"`, ""} {
		stdout, _, err := RunShell(context.Background(), "", "printf %s "+ShellQuote(text), "")
		if err != nil || stdout != text {
			t.Errorf("ShellQuote(%q) round-trips to %q, %v", text, stdout, err)
		}
	}
}
