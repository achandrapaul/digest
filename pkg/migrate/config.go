package migrate

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"

	"github.com/achandrapaul/digest/pkg/config"
	"github.com/achandrapaul/digest/pkg/system"

	"gopkg.in/yaml.v3"
)

var movedJanitorKeys = []struct {
	topLevel string
	option   string
}{
	{"retention_days", config.OptionGraceDays},
	{"janitor_patterns", config.OptionPatterns},
}

func setMappingValue(mapping *yaml.Node, key string, value *yaml.Node, comment string) {
	if index, _ := config.MappingValue(mapping, key); index >= 0 {
		mapping.Content[index+1] = value
		return
	}
	mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key, HeadComment: comment}, value)
}

func builtInJobNodeName(job *yaml.Node) (string, bool) {
	if job.Kind != yaml.MappingNode {
		return "", false
	}
	_, name := config.MappingValue(job, "name")
	if name == nil {
		return "", false
	}
	return config.BuiltInJobName(name.Value)
}

func stringNode(value string) *yaml.Node {
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
	if value == "~" {
		node.Style = yaml.DoubleQuotedStyle
	}
	return node
}

func janitorJobNode(jobs *yaml.Node) *yaml.Node {
	for _, job := range jobs.Content {
		if name, _ := builtInJobNodeName(job); name == config.JobJanitor {
			return job
		}
	}
	janitor := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{stringNode("name"), stringNode(config.JobJanitor)}}
	jobs.Content = append(jobs.Content, janitor)
	return janitor
}

func loadConfigDocument(configPath string) (*yaml.Node, *yaml.Node, error) {
	if configPath == "" {
		return nil, nil, nil
	}
	content, err := system.Read(configPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", configPath, err)
	}
	var document yaml.Node
	if err := yaml.Unmarshal(content, &document); err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", configPath, err)
	}
	if len(document.Content) == 0 {
		return nil, nil, nil
	}
	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("parse %s: the config is not a mapping", configPath)
	}
	return &document, root, nil
}

func saveConfigDocument(configPath string, document *yaml.Node) error {
	var migrated bytes.Buffer
	encoder := yaml.NewEncoder(&migrated)
	encoder.SetIndent(2)
	if err := encoder.Encode(document); err != nil {
		return fmt.Errorf("write %s: %w", configPath, err)
	}
	if err := system.Write(configPath, migrated.Bytes()); err != nil {
		return fmt.Errorf("write %s: %w", configPath, err)
	}
	return nil
}

func migrateJanitorOptions(configPath string, out io.Writer, dryRun bool) error {
	document, root, err := loadConfigDocument(configPath)
	if err != nil || root == nil {
		return err
	}

	var moved []string
	for _, key := range movedJanitorKeys {
		if index, _ := config.MappingValue(root, key.topLevel); index >= 0 {
			moved = append(moved, key.topLevel)
		}
	}
	if len(moved) == 0 {
		return nil
	}

	_, jobs := config.MappingValue(root, "jobs")
	if jobs == nil {
		jobs = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		setMappingValue(root, "jobs", jobs, "")
	}
	if jobs.Kind != yaml.SequenceNode {
		return fmt.Errorf("parse %s: jobs is not a list", configPath)
	}
	janitor := janitorJobNode(jobs)
	_, options := config.MappingValue(janitor, "options")
	if options == nil || options.Kind != yaml.MappingNode {
		options = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		setMappingValue(janitor, "options", options, "")
	}
	for _, key := range movedJanitorKeys {
		index, value := config.MappingValue(root, key.topLevel)
		if index < 0 {
			continue
		}
		if optionIndex, _ := config.MappingValue(options, key.option); optionIndex < 0 {
			setMappingValue(options, key.option, value, root.Content[index].HeadComment)
		}
		root.Content = append(root.Content[:index], root.Content[index+2:]...)
	}

	if dryRun {
		fmt.Fprintf(out, "would move %s into the janitor job's options in %s\n", strings.Join(moved, ", "), configPath)
		return nil
	}
	if err := saveConfigDocument(configPath, document); err != nil {
		return err
	}
	fmt.Fprintf(out, "moved %s into the janitor job's options in %s\n", strings.Join(moved, ", "), configPath)
	return nil
}

const retiredSyncIntervalKey = "git_auto_sync_interval"

func migrateSyncInterval(configPath string, out io.Writer, dryRun bool) error {
	document, root, err := loadConfigDocument(configPath)
	if err != nil || root == nil {
		return err
	}
	index, _ := config.MappingValue(root, retiredSyncIntervalKey)
	if index < 0 {
		return nil
	}
	if dryRun {
		fmt.Fprintf(out, "would remove %s from %s\n", retiredSyncIntervalKey, configPath)
		return nil
	}
	root.Content = append(root.Content[:index], root.Content[index+2:]...)
	if err := saveConfigDocument(configPath, document); err != nil {
		return err
	}
	fmt.Fprintf(out, "removed %s from %s\n", retiredSyncIntervalKey, configPath)
	return nil
}

func templateAutomationNodes() ([]*yaml.Node, error) {
	template, err := config.TemplateNode()
	if err != nil {
		return nil, err
	}
	_, automations := config.MappingValue(template, "automations")
	if automations == nil || automations.Kind != yaml.SequenceNode {
		return nil, errors.New("the config template has no automations list")
	}
	return automations.Content, nil
}

func automationNodeName(automation *yaml.Node) string {
	if automation.Kind != yaml.MappingNode {
		return ""
	}
	if _, name := config.MappingValue(automation, "name"); name != nil {
		return strings.ToLower(strings.TrimSpace(name.Value))
	}
	return ""
}

var offeredOnceAutomations = []string{config.AutomationGoogleCalendar, config.AutomationSummarise}

func migrateAutomations(configPath string, out io.Writer, dryRun bool, alreadyOffered map[string]bool) (map[string]bool, error) {
	document, root, err := loadConfigDocument(configPath)
	if err != nil || root == nil {
		return nil, err
	}
	_, automations := config.MappingValue(root, "automations")
	if automations == nil {
		return nil, nil
	}
	if automations.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("parse %s: automations is not a list", configPath)
	}
	templateNodes, err := templateAutomationNodes()
	if err != nil {
		return nil, err
	}
	templateTypes := map[string]string{}
	offeredTemplates := map[string]*yaml.Node{}
	for _, templateNode := range templateNodes {
		name := automationNodeName(templateNode)
		if _, automationType := config.MappingValue(templateNode, "automation_type"); automationType != nil {
			templateTypes[name] = automationType.Value
		}
		if slices.Contains(offeredOnceAutomations, name) {
			offeredTemplates[name] = templateNode
		}
	}

	var typed []string
	present, prReviewIndex := map[string]bool{}, -1
	for index, automation := range automations.Content {
		name := automationNodeName(automation)
		if slices.Contains(offeredOnceAutomations, name) {
			present[name] = true
		}
		if name == config.AutomationPRReview && prReviewIndex < 0 {
			prReviewIndex = index
		}
		automationType, builtIn := templateTypes[name]
		if !builtIn {
			continue
		}
		if typeIndex, _ := config.MappingValue(automation, "automation_type"); typeIndex >= 0 {
			continue
		}
		nameIndex, _ := config.MappingValue(automation, "name")
		typeEntry := []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "automation_type"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: automationType},
		}
		automation.Content = append(automation.Content[:nameIndex+2], append(typeEntry, automation.Content[nameIndex+2:]...)...)
		typed = append(typed, name)
	}
	var added []string
	for _, name := range offeredOnceAutomations {
		template := offeredTemplates[name]
		if present[name] || alreadyOffered[name] || template == nil {
			continue
		}
		if prReviewIndex < 0 {
			automations.Content = append(automations.Content, template)
		} else {
			automations.Content = append(automations.Content[:prReviewIndex], append([]*yaml.Node{template}, automations.Content[prReviewIndex:]...)...)
			prReviewIndex++
		}
		added = append(added, name)
	}
	if len(typed) == 0 && len(added) == 0 {
		return present, nil
	}

	verb := "added"
	if dryRun {
		verb = "would add"
	} else if err := saveConfigDocument(configPath, document); err != nil {
		return nil, err
	}
	if len(typed) > 0 {
		fmt.Fprintf(out, "%s automation_type to %s in %s\n", verb, strings.Join(typed, ", "), configPath)
	}
	for _, name := range added {
		fmt.Fprintf(out, "%s the %s automation to %s\n", verb, name, configPath)
		present[name] = true
	}
	return present, nil
}

var jobCommandKeys = []string{"command", "dry-run-command", "cmd", "dry_run_command"}

func rootsInCommand(command string) []string {
	var roots []string
	words := strings.Fields(command)
	for index, word := range words {
		switch {
		case (word == "--root" || word == "-root") && index+1 < len(words):
			roots = append(roots, words[index+1])
		case strings.HasPrefix(word, "--root="), strings.HasPrefix(word, "-root="):
			roots = append(roots, word[strings.Index(word, "=")+1:])
		}
	}
	return roots
}

func migrateBuiltInJob(job *yaml.Node, name string) bool {
	changed := false
	if _, nameNode := config.MappingValue(job, "name"); nameNode.Value != name {
		nameNode.Value = name
		changed = true
	}
	var roots []string
	for _, key := range []string{"command", "cmd", "dry-run-command", "dry_run_command"} {
		if _, value := config.MappingValue(job, key); value != nil && len(roots) == 0 {
			roots = rootsInCommand(value.Value)
		}
	}
	for _, key := range jobCommandKeys {
		if index, _ := config.MappingValue(job, key); index >= 0 {
			job.Content = append(job.Content[:index], job.Content[index+2:]...)
			changed = true
		}
	}
	if len(roots) == 0 {
		return changed
	}
	_, options := config.MappingValue(job, "options")
	if options == nil || options.Kind != yaml.MappingNode {
		options = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		setMappingValue(job, "options", options, "")
	}
	if index, _ := config.MappingValue(options, config.OptionRoots); index < 0 {
		rootsNode := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, root := range roots {
			rootsNode.Content = append(rootsNode.Content, stringNode(root))
		}
		setMappingValue(options, config.OptionRoots, rootsNode, "")
	}
	return true
}

func migrateBuiltInJobs(configPath string, out io.Writer, dryRun bool) error {
	document, root, err := loadConfigDocument(configPath)
	if err != nil || root == nil {
		return err
	}
	_, jobs := config.MappingValue(root, "jobs")
	if jobs == nil {
		return nil
	}
	if jobs.Kind != yaml.SequenceNode {
		return fmt.Errorf("parse %s: jobs is not a list", configPath)
	}
	var migrated []string
	for _, job := range jobs.Content {
		if name, builtIn := builtInJobNodeName(job); builtIn && migrateBuiltInJob(job, name) && !slices.Contains(migrated, name) {
			migrated = append(migrated, name)
		}
	}
	if len(migrated) == 0 {
		return nil
	}
	verb := "removed"
	if dryRun {
		verb = "would remove"
	} else if err := saveConfigDocument(configPath, document); err != nil {
		return err
	}
	fmt.Fprintf(out, "%s the commands of %s in %s; their --root values are now the roots option\n", verb, strings.Join(migrated, ", "), configPath)
	return nil
}
