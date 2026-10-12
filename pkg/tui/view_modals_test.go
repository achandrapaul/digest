package tui

import (
	"strings"
	"testing"

	"github.com/achandrapaul/digest/pkg/automation"
	"github.com/achandrapaul/digest/pkg/brag"
	"github.com/achandrapaul/digest/pkg/model"
	"github.com/achandrapaul/digest/pkg/review"
	"github.com/achandrapaul/digest/pkg/sourcecontrol"
)

func TestDeleteConfirmTextsForJobsAndNoteCounts(t *testing.T) {
	m := syncTestModel(t)
	m.mode = ViewDeleteConfirm
	m.jobs.jobToExecute = "nightly"
	if view := stripANSI(m.View()); !strings.Contains(view, "EXECUTE JOB") || !strings.Contains(view, "Are you sure you want to run 'nightly'?") {
		t.Fatalf("execute confirm:\n%s", view)
	}
	m.jobs.jobToExecute = ""
	m.notesUI.deleteTargetNotes = []*model.Note{{Summary: "a"}, {Summary: "b"}}
	if view := stripANSI(m.View()); !strings.Contains(view, "permanently delete these 2 selected notes?") {
		t.Fatalf("multi delete confirm:\n%s", view)
	}
	m.notesUI.deleteTargetNotes = nil
	if view := stripANSI(m.View()); !strings.Contains(view, "No notes selected for deletion.") {
		t.Fatalf("empty delete confirm:\n%s", view)
	}
}

func previewHeader(t *testing.T, m Model) string {
	t.Helper()
	m.mode = ViewPreview
	return stripANSI(m.View())
}

func previewTitleLine(t *testing.T, view, title string) string {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, title) {
			return strings.TrimRight(strings.TrimSpace(line), "│ ")
		}
	}
	t.Fatalf("no %q header in:\n%s", title, view)
	return ""
}

func TestPreviewHeaderForARepoRow(t *testing.T) {
	m := gitStripTestModel(t)
	selectNavKind(t, &m, KindGitRepo)
	m.updatePreviewViewport()
	view := previewHeader(t, m)
	item, _ := m.selectedNavItem()
	if titleLine := previewTitleLine(t, view, "GIT REPO:"); !strings.HasSuffix(titleLine, stripANSI(m.gitRepoStats(item.GitRepo))) || !strings.Contains(titleLine, "commits") {
		t.Fatalf("header should end with the row stats: %q", titleLine)
	}
	if strings.Contains(view, "SYNCED") || strings.Contains(view, "#git") {
		t.Fatalf("old badges should be gone:\n%s", view)
	}
	if !strings.Contains(view, "Repository Activity:") || !strings.Contains(view, "Commits:") {
		t.Fatalf("repo preview body:\n%s", view)
	}
}

func TestPreviewHeaderForAReReviewAndItsNotice(t *testing.T) {
	m := reReviewModel(t, nil)
	m.git.ghPendingPRs[0].Kind = sourcecontrol.ReReviewKind
	m.review.reviewNotice = "Review started in the background"
	view := previewHeader(t, m)
	if !strings.Contains(view, "RE-REVIEW: ") || !strings.Contains(view, "Review started in the background") {
		t.Fatalf("preview:\n%s", view)
	}
	item, _ := m.selectedNavItem()
	if titleLine := previewTitleLine(t, view, "RE-REVIEW: "); !strings.HasSuffix(titleLine, stripANSI(joinTags(m.prTags(item.PendingGitPR, true)))) || strings.Contains(view, "#github") {
		t.Fatalf("header should end with the row tags: %q", titleLine)
	}
}

func TestPreviewHeaderForBragJobs(t *testing.T) {
	m, _ := bragTestModel(t, wednesday())
	m.brag.bragRuns = []brag.Run{{Meta: brag.RunMeta{ID: "2026-W40"}, Status: brag.RunFailed}}
	selectNavKind(t, &m, KindBragRun)
	if view := previewHeader(t, m); !strings.HasSuffix(previewTitleLine(t, view, "BRAG JOB: 2026-W40"), "failed") || strings.Contains(view, "#brag") {
		t.Fatalf("preview:\n%s", view)
	}
	m.brag.bragRuns[0].Status = brag.RunRunning
	if titleLine := previewTitleLine(t, previewHeader(t, m), "BRAG JOB: 2026-W40"); !strings.HasSuffix(titleLine, "bragging...") {
		t.Fatalf("running header: %q", titleLine)
	}
}

func TestPreviewHeaderForJobDryRunStates(t *testing.T) {
	tempDigestRoot(t)
	m, _, _ := jobTestModel(t)
	m.jobs.dryRunsInFlight = map[string]bool{"nightly": true}
	if titleLine := previewTitleLine(t, previewHeader(t, m), "JOB: nightly"); !strings.HasSuffix(titleLine, "#job   dry run...") {
		t.Fatalf("in flight: %q", titleLine)
	}
	m.jobs.dryRunsInFlight = nil
	m.jobs.jobDryRunHasRun = map[string]bool{"nightly": true}
	m.jobs.jobDryRunExitCodes = map[string]int{"nightly": 0}
	if titleLine := previewTitleLine(t, previewHeader(t, m), "JOB: nightly"); !strings.HasSuffix(titleLine, "#job   done") {
		t.Fatalf("success: %q", titleLine)
	}
	m.jobs.jobDryRunExitCodes["nightly"] = 2
	if view := previewHeader(t, m); !strings.HasSuffix(previewTitleLine(t, view, "JOB: nightly"), "#job   act") || strings.Contains(view, "NEED ACT") {
		t.Fatalf("needs action:\n%s", view)
	}
}

func TestRunPreviewHeadersShowTheirRowTag(t *testing.T) {
	cases := []struct {
		item NavItem
		want string
	}{
		{NavItem{Kind: KindReviewRun, ReviewRun: &review.ReviewRun{Status: review.RunFailed}}, "failed"},
		{NavItem{Kind: KindReviewRun, ReviewRun: &review.ReviewRun{Status: review.RunRunning}}, "reviewing..."},
		{NavItem{Kind: KindAutomationRun, AutomationRun: &automation.Run{Status: automation.RunNeedsReauth}}, "needs re-auth"},
		{NavItem{Kind: KindAutomationRun, AutomationRun: &automation.Run{Status: automation.RunRunning, Meta: automation.RunMeta{Phase: automation.PhaseDraft}}}, "drafting..."},
	}
	for _, testCase := range cases {
		header := previewHeading{}
		kindHandlerFor(testCase.item.Kind).header(Model{}, testCase.item, &header)
		if got := stripANSI(joinTags(header.tags)); got != testCase.want || header.subject != "" {
			t.Errorf("tags = %q subject = %q, want %q", got, header.subject, testCase.want)
		}
	}
}

func TestJobPreviewLabelsANewerDryRunAsAnalysis(t *testing.T) {
	tempDigestRoot(t)
	m, _, _ := jobTestModel(t)
	writeLogsFile(t, "nightly.log", "real output\n")
	writeLogsFile(t, "nightly.dryrun.log", "dry output\n")
	writeLogsFile(t, "nightly.dryrun.exit", "0")
	m.refreshDryRunResults()
	m.mode = ViewPreview
	m.updatePreviewViewport()
	if content := stripANSI(m.previewViewport.View()); !strings.Contains(content, "DRY RUN ANALYSIS") {
		t.Fatalf("preview:\n%s", content)
	}
}

func TestOverlayAtSkipsRowsOutsideAndPadsShortLines(t *testing.T) {
	got := overlayAt("ab\ncd", "XY\nZW", -1, 4)
	lines := strings.Split(stripANSI(got), "\n")
	if lines[0] != "abZW" && lines[0] != "ab  ZW" {
		t.Fatalf("lines = %q", lines)
	}
	if lines[1] != "cd" {
		t.Fatalf("the second row should be untouched, got %q", lines[1])
	}
}

func TestErrorPopupDefaultsItsTitle(t *testing.T) {
	m := syncTestModel(t)
	m.mode, m.errorTitle, m.errorLines = ViewError, "", []string{"boom"}
	if view := stripANSI(m.View()); !strings.Contains(view, " ERROR ") || !strings.Contains(view, "boom") {
		t.Fatalf("view:\n%s", view)
	}
}
