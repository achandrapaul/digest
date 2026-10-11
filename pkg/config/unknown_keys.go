package config

import (
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

var legacyKeys = []string{"repo_roots"}

func knownKeys() map[string]bool {
	known := map[string]bool{}
	for _, key := range legacyKeys {
		known[key] = true
	}
	configType := reflect.TypeFor[Config]()
	for index := range configType.NumField() {
		name, _, _ := strings.Cut(configType.Field(index).Tag.Get("yaml"), ",")
		if name != "" && name != "-" {
			known[name] = true
		}
	}
	return known
}

func UnknownKeys(data []byte) []string {
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil || len(document.Content) == 0 {
		return nil
	}
	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil
	}
	known := knownKeys()
	var unknown []string
	for index := 0; index+1 < len(root.Content); index += 2 {
		if key := root.Content[index].Value; !known[key] {
			unknown = append(unknown, key)
		}
	}
	return unknown
}
