package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestTemplateNodeIsAFreshMappingEachCall(t *testing.T) {
	first, err := TemplateNode()
	if err != nil {
		t.Fatal(err)
	}
	if first.Kind != yaml.MappingNode {
		t.Fatalf("template root kind = %v", first.Kind)
	}
	first.Content = nil
	second, err := TemplateNode()
	if err != nil {
		t.Fatal(err)
	}
	if index, automations := MappingValue(second, "automations"); index < 0 || automations.Kind != yaml.SequenceNode {
		t.Fatal("a later call should not see changes to an earlier template")
	}
}

func TestMappingValueFindsKeysOnly(t *testing.T) {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte("first: a\nsecond: first\n"), &document); err != nil {
		t.Fatal(err)
	}
	root := document.Content[0]
	if index, value := MappingValue(root, "second"); index != 2 || value.Value != "first" {
		t.Errorf("second = %d %v", index, value)
	}
	if index, value := MappingValue(root, "a"); index != -1 || value != nil {
		t.Error("values are not keys")
	}
}
