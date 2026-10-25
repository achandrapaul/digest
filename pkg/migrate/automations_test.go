package migrate

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/achandrapaul/digest/pkg/appstate"
	"github.com/achandrapaul/digest/pkg/config"
)

const untypedAutomationsConfig = `digest_root: ~/digest
automations:
  # my jira
  - name: jira
    plugins:
      - jira-inator
    match:
      - make a ticket
    draft_prompt: my own draft
    create_prompt: ""
  - name: docs
    automation_type: doc
    draft_prompt: draft it
    create_prompt: make it
  - name: pr review
    prompt: ""
`

func automationByName(cfg *config.Config, name string) config.AutomationSpec {
	for _, spec := range cfg.AutomationList() {
		if spec.Name == name {
			return spec
		}
	}
	return config.AutomationSpec{}
}

func TestMigrateTypesBuiltInAutomationsAndAddsGoogleCalendar(t *testing.T) {
	f := newFixture(t)
	options, configPath := f.withConfig(t, untypedAutomationsConfig)
	var out bytes.Buffer
	options.Out = &out
	if _, err := Run(options); err != nil {
		t.Fatal(err)
	}
	content, cfg := readConfig(t, configPath)
	if !strings.Contains(content, "  - name: jira\n    automation_type: ticket\n") || !strings.Contains(content, "  - name: pr review\n    automation_type: reviewed\n") {
		t.Errorf("built-in entries should get their type right after the name:\n%s", content)
	}
	jira := automationByName(cfg, config.AutomationJira)
	if jira.DraftPrompt != "my own draft" || strings.Join(jira.Match, ",") != "make a ticket" || !strings.Contains(content, "# my jira") || !strings.Contains(content, `create_prompt: ""`) {
		t.Errorf("migrate should keep your prompts, matches, comments and empty prompts:\n%s", content)
	}
	calendar := automationByName(cfg, config.AutomationGoogleCalendar)
	if calendar.Type != "event" || strings.Join(calendar.Plugins, ",") != "claude_ai_Google_Calendar" || !strings.Contains(string(calendar.DraftPrompt), "NO TIME:") {
		t.Errorf("google calendar = %+v", calendar)
	}
	if strings.Index(content, "name: google calendar") > strings.Index(content, "name: pr review") {
		t.Errorf("google calendar should sit before pr review:\n%s", content)
	}
	if !strings.Contains(out.String(), "added automation_type to jira, pr review in "+configPath) || !strings.Contains(out.String(), "added the google calendar automation to "+configPath) {
		t.Errorf("output = %q", out.String())
	}
	out.Reset()
	if _, err := Run(options); err != nil {
		t.Fatal(err)
	}
	if again, _ := os.ReadFile(configPath); string(again) != content || strings.Contains(out.String(), "automation") {
		t.Errorf("a second migrate should change nothing, output %q", out.String())
	}
}

func TestMigrateDryRunOnlyReportsAutomationChanges(t *testing.T) {
	f := newFixture(t)
	options, configPath := f.withConfig(t, untypedAutomationsConfig)
	var out bytes.Buffer
	options.Out, options.DryRun = &out, true
	if _, err := Run(options); err != nil {
		t.Fatal(err)
	}
	if content, _ := os.ReadFile(configPath); string(content) != untypedAutomationsConfig {
		t.Errorf("a dry run changed the config:\n%s", content)
	}
	if !strings.Contains(out.String(), "would add automation_type to jira, pr review in "+configPath) || !strings.Contains(out.String(), "would add the google calendar automation to "+configPath) {
		t.Errorf("output = %q", out.String())
	}
}

func TestMigrateLeavesAConfigWithoutAutomationsAlone(t *testing.T) {
	f := newFixture(t)
	const noAutomations = "digest_root: ~/digest\n"
	options, configPath := f.withConfig(t, noAutomations)
	if _, err := Run(options); err != nil {
		t.Fatal(err)
	}
	if content, _ := os.ReadFile(configPath); string(content) != noAutomations {
		t.Errorf("built-in defaults already apply, so nothing should be written:\n%s", content)
	}
}

func TestMigrateAddsGoogleCalendarOnlyOnce(t *testing.T) {
	f := newFixture(t)
	options, configPath := f.withConfig(t, untypedAutomationsConfig)
	if _, err := Run(options); err != nil {
		t.Fatal(err)
	}
	if state, _, err := appstate.Load(f.root); err != nil || !state.CalendarOffered {
		t.Fatalf("app state should record the calendar was offered: %+v %v", state, err)
	}
	withoutCalendar := "digest_root: ~/digest\nautomations:\n  - name: jira\n    automation_type: ticket\n    draft_prompt: d\n    create_prompt: c\n"
	if err := os.WriteFile(configPath, []byte(withoutCalendar), 0o600); err != nil {
		t.Fatal(err)
	}
	f.output.Reset()
	if _, err := Run(options); err != nil {
		t.Fatal(err)
	}
	if content, _ := os.ReadFile(configPath); string(content) != withoutCalendar || strings.Contains(f.output.String(), "google calendar") {
		t.Errorf("a deleted calendar should stay deleted, output %q:\n%s", f.output.String(), content)
	}
}

func TestMigrateRecordsACalendarYouAlreadyHave(t *testing.T) {
	f := newFixture(t)
	withCalendar := "digest_root: ~/digest\nautomations:\n  - name: google calendar\n    automation_type: event\n    draft_prompt: d\n    create_prompt: c\n"
	options, _ := f.withConfig(t, withCalendar)
	if _, err := Run(options); err != nil {
		t.Fatal(err)
	}
	if state, _, err := appstate.Load(f.root); err != nil || !state.CalendarOffered {
		t.Errorf("state = %+v, err %v", state, err)
	}
}

func TestMigrateAddsSummariseBeforePRReviewOnlyOnce(t *testing.T) {
	f := newFixture(t)
	options, configPath := f.withConfig(t, untypedAutomationsConfig)
	if _, err := Run(options); err != nil {
		t.Fatal(err)
	}
	content, cfg := readConfig(t, configPath)
	summarise := automationByName(cfg, config.AutomationSummarise)
	if summarise.Type != config.AutomationTypePRSummary || len(summarise.Match) == 0 || summarise.DraftPrompt == "" {
		t.Errorf("summarise = %+v", summarise)
	}
	calendarAt, summariseAt, reviewAt := strings.Index(content, "name: google calendar"), strings.Index(content, "name: summarise"), strings.Index(content, "name: pr review")
	if calendarAt > summariseAt || summariseAt > reviewAt {
		t.Errorf("summarise should sit after google calendar and before pr review:\n%s", content)
	}
	if !strings.Contains(f.output.String(), "added the summarise automation to "+configPath) {
		t.Errorf("output = %q", f.output.String())
	}
	if state, _, err := appstate.Load(f.root); err != nil || !state.SummariseOffered {
		t.Fatalf("app state should record summarise was offered: %+v %v", state, err)
	}
	withoutSummarise := "digest_root: ~/digest\nautomations:\n  - name: jira\n    automation_type: ticket\n    draft_prompt: d\n    create_prompt: c\n"
	if err := os.WriteFile(configPath, []byte(withoutSummarise), 0o600); err != nil {
		t.Fatal(err)
	}
	f.output.Reset()
	if _, err := Run(options); err != nil {
		t.Fatal(err)
	}
	if again, _ := os.ReadFile(configPath); string(again) != withoutSummarise || strings.Contains(f.output.String(), "summarise") {
		t.Errorf("a deleted summarise should stay deleted, output %q:\n%s", f.output.String(), again)
	}
}

func TestMigrateRecordsASummariseYouAlreadyHave(t *testing.T) {
	f := newFixture(t)
	withSummarise := "digest_root: ~/digest\nautomations:\n  - name: summarise\n    automation_type: pr-summary\n    draft_prompt: d\n"
	options, _ := f.withConfig(t, withSummarise)
	if _, err := Run(options); err != nil {
		t.Fatal(err)
	}
	if state, _, err := appstate.Load(f.root); err != nil || !state.SummariseOffered {
		t.Errorf("state = %+v, err %v", state, err)
	}
}
