package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestUnknownKeysListsTopLevelTyposInOrder(t *testing.T) {
	data := []byte("digest_root: ~/digest\ngreen_onyl: true\nrepo_roots: [~/x]\nshow_tag: always\njobs: []\n")
	if got, want := UnknownKeys(data), []string{"green_onyl", "show_tag"}; !reflect.DeepEqual(got, want) {
		t.Errorf("UnknownKeys = %v, want %v", got, want)
	}
}

func TestUnknownKeysIsEmptyForTheTemplateAndForBadYAML(t *testing.T) {
	if got := UnknownKeys([]byte(DefaultConfigYAML)); len(got) != 0 {
		t.Errorf("template has unknown keys %v", got)
	}
	if got := UnknownKeys([]byte(":\n- [")); got != nil {
		t.Errorf("bad yaml = %v, want nil", got)
	}
}

func TestLoadKeepsUnknownKeysForWarnings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("digest_root: /x\ngit_auto_sync_interval: 600\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"git_auto_sync_interval"}; !reflect.DeepEqual(cfg.UnknownKeys, want) {
		t.Errorf("cfg.UnknownKeys = %v, want %v", cfg.UnknownKeys, want)
	}
}
