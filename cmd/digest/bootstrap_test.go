package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/achandrapaul/digest/pkg/appstate"
)

func countingStage(name string, failures int, ran *[]string) bootstrapStage {
	return bootstrapStage{name: name, run: func() error {
		*ran = append(*ran, name)
		if failures > 0 {
			failures--
			return errors.New(name + " broke")
		}
		return nil
	}}
}

func TestRunStagesRetriesAFailedStageOnceAndCarriesOn(t *testing.T) {
	var ran []string
	var out bytes.Buffer
	stages := []bootstrapStage{countingStage("install", 1, &ran), countingStage("doctor", 0, &ran)}
	if err := runStages(stages, &out); err != nil {
		t.Fatal(err)
	}
	if want := []string{"install", "install", "doctor"}; !slices.Equal(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
	if !strings.Contains(out.String(), "install broke") || !strings.Contains(out.String(), "Retrying install") {
		t.Errorf("the first error and the retry should be printed:\n%s", out.String())
	}
}

func TestRunStagesStopsWhenTheRetryFailsToo(t *testing.T) {
	var ran []string
	var out bytes.Buffer
	stages := []bootstrapStage{countingStage("install", 0, &ran), countingStage("doctor", 2, &ran), countingStage("init", 0, &ran)}
	err := runStages(stages, &out)
	if err == nil || !strings.Contains(err.Error(), "doctor") {
		t.Fatalf("err = %v", err)
	}
	if want := []string{"install", "doctor", "doctor"}; !slices.Equal(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
	if strings.Count(out.String(), "doctor broke") != 2 {
		t.Errorf("both errors should be printed:\n%s", out.String())
	}
}

func TestInitStageKeepsAnExistingConfigAndWritesAMissingOne(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	var out bytes.Buffer
	if err := initConfigStage(configPath, &out); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(configPath)
	if err != nil || !strings.Contains(string(written), "digest_root") {
		t.Fatalf("a missing config should be written: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("digest_root: /mine\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := initConfigStage(configPath, &out); err != nil {
		t.Fatal(err)
	}
	if kept, _ := os.ReadFile(configPath); string(kept) != "digest_root: /mine\n" || !strings.Contains(out.String(), "Keeping "+configPath) {
		t.Errorf("an existing config should be kept: %q, output %q", kept, out.String())
	}
}

func TestInitAppStateStageWritesAnEmptyStateAndKeepsAnExistingOne(t *testing.T) {
	root := filepath.Join(t.TempDir(), "digest")
	var out bytes.Buffer
	if err := initAppStateStage(root, &out); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(appstate.Path(root))
	if err != nil || !strings.Contains(string(written), `"first_note_created": null`) || !strings.Contains(string(written), `"streak": 0`) {
		t.Fatalf("a missing state should be written empty: %s, %v", written, err)
	}
	if !strings.Contains(out.String(), "Wrote "+appstate.Path(root)) {
		t.Errorf("output = %q", out.String())
	}
	mine := []byte(`{"first_note_created": "2026-01-02T09:00:00Z", "streak": 4}`)
	if err := os.WriteFile(appstate.Path(root), mine, 0o600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := initAppStateStage(root, &out); err != nil {
		t.Fatal(err)
	}
	if kept, _ := os.ReadFile(appstate.Path(root)); !bytes.Equal(kept, mine) || !strings.Contains(out.String(), "Keeping "+appstate.Path(root)) {
		t.Errorf("an existing state should be kept: %s, output %q", kept, out.String())
	}
}

func TestBootstrapRefusesDryRun(t *testing.T) {
	if _, refused := dryRunUnsupported("bootstrap", nil); !refused {
		t.Error("bootstrap should refuse --dry-run")
	}
}
