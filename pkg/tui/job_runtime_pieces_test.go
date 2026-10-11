package tui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/model"
)

var realStartJobProcess = startJobProcess

func tempDigestRoot(t *testing.T) string {
	t.Helper()
	previousRoot := currentDigestRoot()
	setDigestRoot(t.TempDir())
	t.Cleanup(func() {
		setDigestRoot(previousRoot)
	})
	return jobRuntime().LogsDir
}

func writeLogsFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(jobRuntime().LogsDir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
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

func TestReadFileTailOfADirectoryIsEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "entry"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := readFileTail(dir, 1024); got != "" {
		t.Fatalf("tail = %q", got)
	}
}

func TestDryRunOutputIsShownOnlyAfterARun(t *testing.T) {
	m := syncTestModel(t)
	m.jobs.jobDryRunHasRun = map[string]bool{"nightly": true}
	m.jobs.jobDryRunOutputs = map[string]string{"nightly": "checked"}
	if m.jobDryRunOutputFor("nightly") != "checked" {
		t.Fatal("a finished dry run shows its output")
	}
}

func TestPreviewedRunningJobOnlyForJobPreviews(t *testing.T) {
	tempDigestRoot(t)
	m, _, _ := jobTestModel(t)
	m.notes = []*model.Note{{ID: "n", Summary: "note", Source: model.SourceManual, Status: model.StatusActive, Created: m.currentDate, Updated: m.currentDate}}
	m.contentVersion++
	selectNavItem(t, &m, "job:nightly")
	if m.previewedRunningJob() != "" {
		t.Fatal("the dashboard previews no job")
	}
	m.mode = ViewPreview
	if m.previewedRunningJob() != "nightly" {
		t.Fatalf("job = %q", m.previewedRunningJob())
	}
	m.selected = 999
	if m.previewedRunningJob() != "" {
		t.Fatal("an out of range selection previews no job")
	}
	selectNavKind(t, &m, KindTodayNote)
	if m.previewedRunningJob() != "" {
		t.Fatal("a non-job row previews no job")
	}
}

func TestDryRunStartFailureShowsAJobError(t *testing.T) {
	tempDigestRoot(t)
	m, _, _ := jobTestModel(t)
	startJobProcess = func(launch jobLaunch) error {
		if launch.dryRun {
			return errors.New("no shell")
		}
		return nil
	}
	next, followUp := m.applyJobLaunch(m.startJobDryRunCmd("nightly")().(jobLaunchedMsg))
	if m = next.(Model); followUp != nil || latestMessageText(m) != "no shell" {
		t.Fatalf("message %q", latestMessageText(m))
	}
}
