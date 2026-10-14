package sourcecontrol

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/config"
	"github.com/achandrapaul/digest/pkg/review"
)

func makeRepo(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(path, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestConfiguredRepoNames(t *testing.T) {
	base := t.TempDir()
	oneControl := makeRepo(t, filepath.Join(base, "github.com", "o", "web-control"))
	makeRepo(t, filepath.Join(base, "github.com", "o", "unlisted"))
	projects := filepath.Join(base, "projects")
	makeRepo(t, filepath.Join(projects, "monkey"))
	cfg := &config.Config{GitRepositoryRoots: []string{oneControl + "/", projects}}
	names := ConfiguredRepoNames(cfg)
	if !names["web-control"] || !names["monkey"] || names["unlisted"] || len(names) != 2 {
		t.Errorf("names = %v", names)
	}
	if ConfiguredRepoNames(&config.Config{}) != nil {
		t.Errorf("no roots should mean no filter")
	}
}

func TestFilterPRItemsAndApprovals(t *testing.T) {
	allowed := map[string]bool{"control": true}
	items := []PRItem{{Repository: "control"}, {Repository: "monkey"}, {Repository: "acme/control"}}
	if kept := FilterPRItems(items, allowed); len(kept) != 2 || kept[0].Repository != "control" || kept[1].Repository != "acme/control" {
		t.Errorf("kept = %+v", kept)
	}
	if kept := FilterPRItems(items, nil); len(kept) != 3 {
		t.Errorf("nil filter dropped items: %d", len(kept))
	}
	approvals := []review.ActivityPR{{Repository: "control", Number: 1}, {Repository: "monkey", Number: 2}}
	if kept := FilterReviewRecords(approvals, allowed); len(kept) != 1 || kept[0].Number != 1 {
		t.Errorf("approvals = %+v", kept)
	}
	if kept := FilterReviewRecords(approvals, nil); len(kept) != 2 {
		t.Errorf("nil filter dropped approvals")
	}
}

func TestLocalRepoPathsUsesOnlyRoots(t *testing.T) {
	base := t.TempDir()
	listed := makeRepo(t, filepath.Join(base, "listed"))
	makeRepo(t, filepath.Join(base, "other"))
	cfg := &config.Config{GitRepositoryRoots: []string{listed}, DigestRoot: base}
	paths := localRepoPaths(cfg)
	if len(paths) != 1 || filepath.Base(paths[0]) != "listed" {
		t.Errorf("paths = %v", paths)
	}
	if fallback := localRepoPaths(&config.Config{DigestRoot: base}); len(fallback) != 0 {
		t.Errorf("no roots should search nowhere, got %v", fallback)
	}
}

func TestDayResultOutsideRootsIsFiltered(t *testing.T) {
	cfg := &config.Config{GitRepositoryRoots: []string{makeRepo(t, filepath.Join(t.TempDir(), "control"))}}
	result := filterDayResult(DayResult{
		Day:      Today,
		Reviewed: []PRItem{{Repository: "control", Kind: ReviewedKind}, {Repository: "monkey", Kind: ReviewedKind}},
		Reviews:  []review.ActivityPR{{Number: 1, Repository: "control", State: "APPROVED", ReviewedAt: time.Now()}, {Number: 2, Repository: "monkey", State: "APPROVED", ReviewedAt: time.Now()}},
	}, ConfiguredRepoNames(cfg))
	if len(result.Reviewed) != 1 || len(result.Reviews) != 1 || result.Reviews[0].Repository != "control" {
		t.Errorf("filtered = %+v", result)
	}
}

func TestLocalRepoPathsReusesTheRecentWalk(t *testing.T) {
	root := t.TempDir()
	makeRepo(t, filepath.Join(root, "first"))
	cfg := &config.Config{GitRepositoryRoots: []string{root}}
	if paths := localRepoPaths(cfg); len(paths) != 1 {
		t.Fatalf("paths = %v", paths)
	}
	makeRepo(t, filepath.Join(root, "second"))
	if paths := localRepoPaths(cfg); len(paths) != 1 {
		t.Errorf("a refresh within the cache window should not walk the roots again: %v", paths)
	}
}
