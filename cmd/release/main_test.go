package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func gitIn(t *testing.T, repoDir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = repoDir
	command.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func commitFile(t *testing.T, repoDir, name, content, subject string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repoDir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repoDir, "add", name)
	gitIn(t, repoDir, "commit", "-q", "-m", subject)
}

func subjectsForRelease(t *testing.T, repoDir, version string) []string {
	t.Helper()
	commits, err := commitsForRelease(repoDir, version)
	if err != nil {
		t.Fatal(err)
	}
	var subjects []string
	for _, commit := range commits {
		subjects = append(subjects, commit.Subject)
	}
	return subjects
}

func TestCommitsForReleaseFollowTheVersionTags(t *testing.T) {
	repoDir := t.TempDir()
	gitIn(t, repoDir, "init", "-q")
	commitFile(t, repoDir, "a.go", "a", "feat: first")
	if got := subjectsForRelease(t, repoDir, "0.0.1"); !slices.Equal(got, []string{"feat: first"}) {
		t.Errorf("without a tag every commit is listed, got %v", got)
	}
	commitFile(t, repoDir, ".version", "0.0.1\n", "release:patch")
	gitIn(t, repoDir, "tag", "-m", "t", "v0.0.1")
	commitFile(t, repoDir, "CHANGELOG.md", "x", "release:patch")
	commitFile(t, repoDir, "a.go", "b", "fix: second")
	commitFile(t, repoDir, ".version", "0.0.2\n", "release:patch")
	gitIn(t, repoDir, "tag", "-m", "t", "v0.0.2")
	commitFile(t, repoDir, "b.go", "c", "feat: third")

	cases := map[string][]string{
		"0.0.1": {"release:patch", "feat: first"},
		"0.0.2": {"release:patch", "fix: second", "release:patch"},
		"0.0.3": {"feat: third"},
		"":      {"feat: third"},
	}
	for version, want := range cases {
		if got := subjectsForRelease(t, repoDir, version); !slices.Equal(got, want) {
			t.Errorf("version %q: got %v, want %v", version, got, want)
		}
	}
}

func TestVersionTagsSortNewestVersionFirst(t *testing.T) {
	repoDir := t.TempDir()
	gitIn(t, repoDir, "init", "-q")
	commitFile(t, repoDir, "a.go", "a", "feat: first")
	for _, tag := range []string{"v0.0.9", "v0.0.10", "v0.0.2", "other"} {
		gitIn(t, repoDir, "tag", "-m", "t", tag)
	}
	if got, want := versionTags(repoDir), []string{"v0.0.10", "v0.0.9", "v0.0.2"}; !slices.Equal(got, want) {
		t.Errorf("versionTags = %v, want %v", got, want)
	}
}

func TestNotesForATaggedVersionUseTheTaggedCommitDateInUTC(t *testing.T) {
	repoDir := t.TempDir()
	gitIn(t, repoDir, "init", "-q")
	if err := os.WriteFile(filepath.Join(repoDir, "a.go"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repoDir, "add", "a.go")
	command := exec.Command("git", "commit", "-q", "-m", "feat: first")
	command.Dir = repoDir
	command.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t", "GIT_COMMITTER_DATE=2026-10-11T03:55:13+05:30")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("commit: %v\n%s", err, output)
	}
	gitIn(t, repoDir, "tag", "-m", "t", "v0.0.1")
	if err := writeNotes(repoDir, "0.0.1", io.Discard); err != nil {
		t.Fatal(err)
	}
	changelog, err := os.ReadFile(filepath.Join(repoDir, changelogPath))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(changelog), "# Changelog\n\n## v0.0.1 — 2026-10-10\n") || !strings.Contains(string(changelog), ") feat: first\n") {
		t.Errorf("changelog =\n%s", changelog)
	}
}

func TestNotesWritesTheReleaseSectionOnceAndPrintsIt(t *testing.T) {
	repoDir := t.TempDir()
	gitIn(t, repoDir, "init", "-q")
	commitFile(t, repoDir, ".version", "1.4.0\n", "chore: release v1.4.0")
	commitFile(t, repoDir, "a.go", "a", "fix: second")
	changelogFile := filepath.Join(repoDir, changelogPath)
	if err := os.WriteFile(changelogFile, []byte("# Changelog\n\n## v1.4.0 — 2026-10-01\n\n### Changes\n- old\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var firstRun, secondRun strings.Builder
	if err := writeNotes(repoDir, "1.4.1", &firstRun); err != nil {
		t.Fatal(err)
	}
	if err := writeNotes(repoDir, "1.4.1", &secondRun); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(firstRun.String(), "### Changes\n- [") || !strings.Contains(firstRun.String(), ") fix: second\n") {
		t.Errorf("notes = %q", firstRun.String())
	}
	if secondRun.String() != firstRun.String() {
		t.Errorf("second run printed %q, want %q", secondRun.String(), firstRun.String())
	}
	changelog, err := os.ReadFile(changelogFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(changelog), "## v1.4.1 — ") != 1 || !strings.Contains(string(changelog), "## v1.4.0 — 2026-10-01") {
		t.Errorf("changelog =\n%s", changelog)
	}
	if !strings.Contains(string(changelog), "](https://github.com/achandrapaul/digest/commit/") {
		t.Errorf("changelog should link each commit:\n%s", changelog)
	}
}
