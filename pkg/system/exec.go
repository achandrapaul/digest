package system

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

var CommandWaitDelay = 2 * time.Second

func KillGroupOnCancel(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
}

func ShellCommand(ctx context.Context, command string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.WaitDelay = CommandWaitDelay
	KillGroupOnCancel(cmd)
	return cmd
}

func RunShell(ctx context.Context, dir, command, stdin string) (string, string, error) {
	cmd := ShellCommand(ctx, command)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func GitCommand(ctx context.Context, dir string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.WaitDelay = CommandWaitDelay
	cmd.Dir = dir
	cmd.Env = append(append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_SSH_COMMAND=ssh -o BatchMode=yes -o ConnectTimeout=5",
	), GitSafeEnv()...)
	return cmd
}

func GHCommand(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.WaitDelay = CommandWaitDelay
	return cmd
}

func ShellQuote(text string) string {
	return "'" + strings.ReplaceAll(text, "'", `'\''`) + "'"
}
