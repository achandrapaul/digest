package main

import (
	"reflect"
	"testing"

	"github.com/achandrapaul/digest/pkg/config"

	"gopkg.in/yaml.v3"
)

func configForTest(t *testing.T, raw string) *config.Config {
	t.Helper()
	var cfg config.Config
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatal(err)
	}
	return &cfg
}

func TestJanitorSettingsComeFromTheJanitorJobOptions(t *testing.T) {
	cfg := configForTest(t, "retention_days: 3\njanitor_patterns: [old.log]\njobs:\n  - name: janitor\n    options:\n      grace_days: 21\n      patterns: [a.log, b.tmp]\n")
	patterns, graceDays, err := janitorSettings(cfg)
	if err != nil || graceDays != 21 || !reflect.DeepEqual(patterns, []string{"a.log", "b.tmp"}) {
		t.Errorf("patterns = %v, grace = %d, err = %v", patterns, graceDays, err)
	}

	t.Setenv("DIGEST_OPTION_GRACE_DAYS", "2")
	t.Setenv("DIGEST_OPTION_PATTERNS", "env.log")
	patterns, graceDays, err = janitorSettings(cfg)
	if err != nil || graceDays != 2 || !reflect.DeepEqual(patterns, []string{"env.log"}) {
		t.Errorf("env: patterns = %v, grace = %d, err = %v", patterns, graceDays, err)
	}

	t.Setenv("DIGEST_OPTION_GRACE_DAYS", "")
	t.Setenv("DIGEST_OPTION_PATTERNS", "")
	patterns, graceDays, err = janitorSettings(configForTest(t, "jobs: []\n"))
	if err != nil || graceDays != config.DefaultJanitorGraceDays || !reflect.DeepEqual(patterns, config.DefaultJanitorPatterns) {
		t.Errorf("defaults: patterns = %v, grace = %d, err = %v", patterns, graceDays, err)
	}

	t.Setenv("DIGEST_OPTION_GRACE_DAYS", "later")
	if _, _, err := janitorSettings(cfg); err == nil {
		t.Error("an unparseable grace_days should be an error")
	}
}

func TestBranchReaperGraceDaysComeFromItsJobOptions(t *testing.T) {
	cfg := configForTest(t, "jobs:\n  - name: branch-reaper\n    options:\n      grace_days: 3\n")
	if days, err := branchReaperGraceDays(cfg); err != nil || days != 3 {
		t.Errorf("config: %d, %v", days, err)
	}
	if days, err := branchReaperGraceDays(configForTest(t, "jobs: []\n")); err != nil || days != config.DefaultBranchReaperGraceDays {
		t.Errorf("default: %d, %v", days, err)
	}
	t.Setenv("DIGEST_OPTION_GRACE_DAYS", "1")
	if days, err := branchReaperGraceDays(cfg); err != nil || days != 1 {
		t.Errorf("env: %d, %v", days, err)
	}
}

func TestJobRootsComeFromTheFlagThenTheEnvThenTheConfig(t *testing.T) {
	cfg := configForTest(t, "jobs:\n  - name: janitor\n    options:\n      roots: [~, ~/Projects/]\n")
	if got := jobRoots(cfg, config.JobJanitor, nil); !reflect.DeepEqual(got, []string{"~", "~/Projects/"}) {
		t.Errorf("config roots = %v", got)
	}
	if got := jobRoots(cfg, config.JobRepoSync, nil); !reflect.DeepEqual(got, []string{"~/Projects"}) {
		t.Errorf("default repo-sync roots = %v", got)
	}
	if got := jobRoots(nil, config.JobJanitor, nil); !reflect.DeepEqual(got, []string{"~"}) {
		t.Errorf("no config janitor roots = %v", got)
	}
	t.Setenv("DIGEST_OPTION_ROOTS", "~/a,~/b")
	if got := jobRoots(cfg, config.JobJanitor, nil); !reflect.DeepEqual(got, []string{"~/a", "~/b"}) {
		t.Errorf("a dashboard run passes roots in the env: %v", got)
	}
	var roots rootFlags
	for _, root := range []string{"~/Projects", "/tmp/work"} {
		if err := roots.Set(root); err != nil {
			t.Fatal(err)
		}
	}
	if got := jobRoots(cfg, config.JobJanitor, roots); !reflect.DeepEqual(got, []string{"~/Projects", "/tmp/work"}) || roots.String() != "~/Projects,/tmp/work" {
		t.Errorf("--root overrides: %v", got)
	}
}
