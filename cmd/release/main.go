package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/achandrapaul/digest/pkg/release"
	"github.com/achandrapaul/digest/pkg/system"
)

const (
	changelogPath = "CHANGELOG.md"
	repoURL       = "https://github.com/achandrapaul/digest"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "release:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: release level | notes <version>")
	}
	switch args[0] {
	case "level":
		commits, err := commitsForRelease(".", "")
		if err != nil {
			return err
		}
		fmt.Println(release.NextLevel(commits))
		return nil
	case "notes":
		if len(args) != 2 {
			return errors.New("usage: release notes <version>")
		}
		return writeNotes(".", args[1], os.Stdout)
	}
	return fmt.Errorf("unknown command %q", args[0])
}

func writeNotes(repoDir, version string, out io.Writer) error {
	changelogFile := filepath.Join(repoDir, changelogPath)
	existing, err := system.Read(changelogFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	changelog := string(existing)
	if !strings.Contains(changelog, "\n## v"+version+" — ") {
		commits, err := commitsForRelease(repoDir, version)
		if err != nil {
			return err
		}
		changelog = release.Prepend(changelog, release.Changelog(version, releaseDate(repoDir, version), repoURL, commits))
		if err := system.WriteWithMode(changelogFile, []byte(changelog), 0o644); err != nil {
			return err
		}
	}
	_, err = fmt.Fprint(out, release.LatestSection(changelog))
	return err
}

func gitOutput(repoDir string, args ...string) (string, error) {
	command := exec.Command("git", args...)
	command.Dir = repoDir
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output)), nil
}

func versionTags(repoDir string) []string {
	output, err := gitOutput(repoDir, "tag", "--list", "v*", "--sort=-v:refname")
	if err != nil || output == "" {
		return nil
	}
	return strings.Split(output, "\n")
}

func releaseRange(repoDir, version string) (string, string) {
	from, tagged := release.PreviousTag(versionTags(repoDir), version)
	if tagged {
		return from, "v" + version
	}
	return from, "HEAD"
}

func releaseDate(repoDir, version string) time.Time {
	if _, to := releaseRange(repoDir, version); to != "HEAD" {
		if tagDate, err := gitOutput(repoDir, "log", "-1", "--format=%cI", to); err == nil {
			if parsed, err := time.Parse(time.RFC3339, tagDate); err == nil {
				return parsed.UTC()
			}
		}
	}
	return time.Now().UTC()
}

func commitsForRelease(repoDir, version string) ([]release.Commit, error) {
	from, to := releaseRange(repoDir, version)
	revisions := to
	if from != "" {
		revisions = from + ".." + to
	}
	logArgs := []string{"log", "--format=%H%x1f%s%x1f%b%x1e", revisions}
	command := exec.Command("git", logArgs...)
	command.Dir = repoDir
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}
	var commits []release.Commit
	for record := range strings.SplitSeq(string(output), "\x1e") {
		hash, rest, _ := strings.Cut(strings.TrimLeft(record, "\n"), "\x1f")
		subject, body, _ := strings.Cut(rest, "\x1f")
		if subject != "" {
			commits = append(commits, release.Commit{Hash: hash, Subject: subject, Body: body})
		}
	}
	return commits, nil
}
