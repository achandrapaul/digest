package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadConfigText(t *testing.T, text string) (*Config, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return Load(path)
}

func TestLoadRejectsMalformedJanitorPatterns(t *testing.T) {
	_, err := loadConfigText(t, "digest_root: /x\njobs:\n  - name: janitor\n    options:\n      patterns: [\"*.hprof\", \"[bad\"]\n")
	if err == nil || !strings.Contains(err.Error(), `janitor option patterns: "[bad"`) {
		t.Fatalf("err = %v", err)
	}
	if _, err := loadConfigText(t, "digest_root: /x\njobs:\n  - name: janitor\n    options:\n      patterns: [\"*\", \"hs_err_pid*.log\"]\n"); err != nil {
		t.Errorf("well-formed patterns should load: %v", err)
	}
}

func TestLoadRejectsBadGraceDays(t *testing.T) {
	for _, job := range []string{JobJanitor, JobBranchReaper} {
		_, err := loadConfigText(t, "digest_root: /x\njobs:\n  - name: "+job+"\n    options:\n      grace_days: soon\n")
		if err == nil || !strings.Contains(err.Error(), job+" option grace_days") {
			t.Errorf("%s: err = %v", job, err)
		}
	}
}

func TestValidatePatterns(t *testing.T) {
	if err := ValidatePatterns([]string{"*.log", "a?b"}); err != nil {
		t.Errorf("valid patterns: %v", err)
	}
	if err := ValidatePatterns([]string{"ok", "x[", "y["}); err == nil || !strings.Contains(err.Error(), `"x["`) {
		t.Errorf("err = %v", err)
	}
}
