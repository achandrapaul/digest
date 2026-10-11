package config

import (
	"errors"

	"gopkg.in/yaml.v3"
)

func TemplateNode() (*yaml.Node, error) {
	var template yaml.Node
	if err := yaml.Unmarshal([]byte(DefaultConfigYAML), &template); err != nil {
		return nil, err
	}
	if len(template.Content) == 0 || template.Content[0].Kind != yaml.MappingNode {
		return nil, errors.New("the config template is not a mapping")
	}
	return template.Content[0], nil
}

func MappingValue(mapping *yaml.Node, key string) (int, *yaml.Node) {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			return index, mapping.Content[index+1]
		}
	}
	return -1, nil
}
