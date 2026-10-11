package jobs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitCommandsIgnoreTheRepoFsmonitorAndHooks(t *testing.T) {
	repo, marker := t.TempDir(), filepath.Join(t.TempDir(), "ran")
	script := filepath.Join(t.TempDir(), "evil.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntouch "+marker+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "init", "--quiet", "-b", "main")
	if err := os.WriteFile(filepath.Join(repo, "README"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "README")
	runGit(t, repo, "commit", "--quiet", "-m", "init")
	runGit(t, repo, "config", "core.fsmonitor", script)
	hooksDir := filepath.Join(repo, ".git", "hooks")
	if err := os.WriteFile(filepath.Join(hooksDir, "reference-transaction"), []byte("#!/bin/sh\ntouch "+marker+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(repo, "status", "--porcelain")
	if result := gitCmd(repo, "branch", "side"); !result.Ok {
		t.Fatalf("git branch failed: %s", result.Stderr)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("a repo-configured fsmonitor or hook ran: %v", err)
	}
}
