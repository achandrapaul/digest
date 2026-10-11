package tui

import (
	"strings"
	"testing"

	"github.com/achandrapaul/digest/pkg/config"
)

func TestJobsWithoutADryRunHideD(t *testing.T) {
	m, _, dryRun := jobTestModel(t)
	m.cfg.Jobs = []config.JobSpec{{Name: "nightly", Command: "true"}, {Name: "janitor", Command: "digest janitor"}, {Name: "branch-reaper", Command: "digest branch-reaper", DryRunCommand: "digest branch-reaper --dry-run"}}
	m.contentVersion++
	selectNavItem(t, &m, "job:nightly")
	if hint := hintText(m); strings.Contains(hint, "(d)") {
		t.Errorf("a job with no dry-run command offers d: %q", hint)
	}
	if m = press(t, m, runes("d")); len(*dryRun) != 0 {
		t.Errorf("d started a dry run for a job without one: %v", *dryRun)
	}
	selectNavItem(t, &m, "job:janitor")
	if hint := hintText(m); strings.Contains(hint, "(d)") {
		t.Errorf("a built-in job with no dry-run command offers d: %q", hint)
	}
	if m = press(t, m, runes("d")); len(*dryRun) != 0 {
		t.Errorf("d started a dry run for a built-in job without one: %v", *dryRun)
	}
	selectNavItem(t, &m, "job:branch-reaper")
	if hint := hintText(m); !strings.Contains(hint, "(d)dry run") {
		t.Errorf("a job with a dry-run command offers d: %q", hint)
	}
}
