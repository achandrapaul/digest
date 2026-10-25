package automation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/achandrapaul/digest/pkg/config"
	"github.com/achandrapaul/digest/pkg/model"
	"github.com/achandrapaul/digest/pkg/review"
	"github.com/achandrapaul/digest/pkg/store"
)

func TestReplaceSectionReplacesOnlyThatSection(t *testing.T) {
	body := "intro\n\n## PR summary\n\nold\n\n### acme/api#1 Old\n\nold detail\n\n## Ticket\n\nPROJ-1"
	got := replaceSection(body, "## PR summary", "## PR summary\n\nnew")
	if got != "intro\n\n## PR summary\n\nnew\n\n## Ticket\n\nPROJ-1" {
		t.Errorf("replaced =\n%s", got)
	}
	if got := replaceSection("intro\n", "## PR summary", "## PR summary\n\nnew"); got != "intro\n\n## PR summary\n\nnew" {
		t.Errorf("appended =\n%s", got)
	}
	if got := replaceSection("", "## PR summary", "## PR summary\n\nnew"); got != "## PR summary\n\nnew" {
		t.Errorf("empty body =\n%s", got)
	}
	if got := replaceSection("see ## PR summary inline", "## PR summary", "## PR summary\n\nnew"); got != "see ## PR summary inline\n\n## PR summary\n\nnew" {
		t.Errorf("only a heading line counts:\n%s", got)
	}
}

func TestCreateUsesTheDraftItIsGiven(t *testing.T) {
	root := t.TempDir()
	inputPath := filepath.Join(root, "input.txt")
	spec := fakeSpec(`cat > '` + inputPath + `'; echo '{"kind":"ticket","key":"PROJ-9"}'`)
	result, err := execute(context.Background(), spec, root, testNote(), PhaseCreate, "summary: Lost logs\n", nil)
	if err != nil || result.Key != "PROJ-9" {
		t.Fatalf("create with a draft in hand and none on disk = %+v, %v", result, err)
	}
	if input, _ := os.ReadFile(inputPath); !strings.Contains(string(input), "CREATE PROMPT\n\n## Draft\n\nsummary: Lost logs") {
		t.Errorf("input =\n%s", input)
	}
}

func summariseSpec(command string) config.AutomationSpec {
	return config.AutomationSpec{Name: "summarise", Type: config.AutomationTypePRSummary, Match: []string{"check pr"}, DraftPrompt: "SUMMARISE", Command: command}
}

func TestSummariseIsOfferedOnlyWithAPullRequestLinkOnAGHHost(t *testing.T) {
	specs := []config.AutomationSpec{summariseSpec("cat"), fakeSpec("cat")}
	ghHosts := []string{"github.com", "code.example.com"}
	for _, testCase := range []struct {
		summary, body string
		want          []string
	}{
		{"check pr", "https://github.com/acme/api/pull/3", []string{"summarise"}},
		{"Check PR https://github.com/acme/api/pull/3 then create jira", "", []string{"summarise", "PROJ"}},
		{"check prices", "", nil},
		{"check pr", "https://github.com/acme/api/issues/3", nil},
		{"create jira", "https://github.com/acme/api/pull/3", []string{"PROJ"}},
		{"check pr", "https://code.example.com/acme/api/pull/3", []string{"summarise"}},
		{"check pr", "https://gitlab.example.com/acme/api/pull/3", nil},
	} {
		var names []string
		for _, spec := range MatchAll(specs, &model.Note{Summary: testCase.summary, Body: testCase.body}, ghHosts) {
			names = append(names, spec.Name)
		}
		if strings.Join(names, ",") != strings.Join(testCase.want, ",") {
			t.Errorf("MatchAll(%q, %q) = %v, want %v", testCase.summary, testCase.body, names, testCase.want)
		}
	}
}

func TestSummariseIsHiddenUntilTheGHHostsAreKnown(t *testing.T) {
	specs := []config.AutomationSpec{summariseSpec("cat"), fakeSpec("cat")}
	note := &model.Note{Summary: "check pr https://github.com/acme/api/pull/3 then create jira"}
	var names []string
	for _, spec := range MatchAll(specs, note, nil) {
		names = append(names, spec.Name)
	}
	if strings.Join(names, ",") != "PROJ" {
		t.Errorf("without gh hosts = %v, want only PROJ", names)
	}
}

func fakeGHHosts(t *testing.T, hosts []string, err error) *int {
	t.Helper()
	calls := 0
	original := lookupGHHosts
	lookupGHHosts = func(context.Context) ([]string, error) {
		calls++
		return hosts, err
	}
	t.Cleanup(func() { lookupGHHosts = original })
	return &calls
}

func TestSummariseDraftSkipsLinksOffTheGHHosts(t *testing.T) {
	fetched := fakeFetchPRContext(t, nil)
	calls := fakeGHHosts(t, []string{"github.com"}, nil)
	note := &model.Note{ID: "n", Summary: "check pr https://github.com/acme/api/pull/1 https://gitlab.example.com/x/y/pull/2 https://github.com/acme/web/pull/3"}
	if _, err := Execute(context.Background(), summariseSpec(`echo '{"summary":"x"}'`), t.TempDir(), note, PhaseDraft, nil); err != nil {
		t.Fatal(err)
	}
	if *calls != 1 || len(*fetched) != 2 || (*fetched)[0].Number != 1 || (*fetched)[1].Number != 3 {
		t.Errorf("gh host lookups = %d, fetched = %+v", *calls, *fetched)
	}
}

func TestSummariseDraftFailsWhenGHHostsAreUnknown(t *testing.T) {
	fakeFetchPRContext(t, nil)
	fakeGHHosts(t, nil, errors.New("gh auth status lists no logged-in hosts"))
	note := &model.Note{ID: "n", Summary: "check pr https://github.com/acme/api/pull/1"}
	if _, err := Execute(context.Background(), summariseSpec(`echo '{"summary":"x"}'`), t.TempDir(), note, PhaseDraft, nil); err == nil || !strings.Contains(err.Error(), "no logged-in hosts") {
		t.Errorf("err = %v", err)
	}
	fakeGHHosts(t, []string{"code.example.com"}, nil)
	if _, err := Execute(context.Background(), summariseSpec(`echo '{"summary":"x"}'`), t.TempDir(), note, PhaseDraft, nil); err == nil || !strings.Contains(err.Error(), "no pull request link on a host gh is logged in to (code.example.com)") {
		t.Errorf("err = %v", err)
	}
}

func fakeFetchPRContext(t *testing.T, failing map[int]error) *[]review.PRRef {
	t.Helper()
	var fetched []review.PRRef
	original := fetchPRContext
	fetchPRContext = func(ctx context.Context, ref review.PRRef, maxDiffBytes int) (string, error) {
		fetched = append(fetched, ref)
		if err := failing[ref.Number]; err != nil {
			return "", err
		}
		return "### " + ref.NameWithOwner() + "#" + strconv.Itoa(ref.Number) + " details\n", nil
	}
	t.Cleanup(func() { fetchPRContext = original })
	fakeGHHosts(t, []string{"github.com"}, nil)
	return &fetched
}

func TestSummariseDraftSendsEveryPullRequest(t *testing.T) {
	root := t.TempDir()
	fetched := fakeFetchPRContext(t, map[int]error{2: errors.New("HTTP 404")})
	inputPath := filepath.Join(root, "input.txt")
	spec := summariseSpec(`cat > '` + inputPath + `'; echo '{"summary":"acme PRs","description":"### acme/api#1 One"}'`)
	note := &model.Note{ID: "note-1", Summary: "check pr https://github.com/acme/api/pull/1",
		Body: "and https://github.com/acme/web/pull/2/files plus https://github.com/acme/api/pull/1 again"}
	if _, err := Execute(context.Background(), spec, root, note, PhaseDraft, nil); err != nil {
		t.Fatal(err)
	}
	if len(*fetched) != 2 || (*fetched)[0].URL != "https://github.com/acme/api/pull/1" || (*fetched)[1].URL != "https://github.com/acme/web/pull/2" {
		t.Errorf("fetched = %+v", *fetched)
	}
	input, _ := os.ReadFile(inputPath)
	for _, want := range []string{"SUMMARISE", "## Note", "## Pull requests", "### acme/api#1 details", "### acme/web#2\n\nUnavailable: HTTP 404"} {
		if !strings.Contains(string(input), want) {
			t.Errorf("input misses %q:\n%s", want, input)
		}
	}
	if draft, err := LoadDraft(root, note.ID); err != nil || !strings.Contains(draft, "acme PRs") {
		t.Errorf("draft = %q, %v", draft, err)
	}
}

func TestSummariseDraftFailsWithoutAnyFetchedPullRequest(t *testing.T) {
	fakeFetchPRContext(t, map[int]error{1: errors.New("HTTP 404")})
	spec := summariseSpec(`echo '{"summary":"x"}'`)
	if _, err := Execute(context.Background(), spec, t.TempDir(), &model.Note{ID: "n", Summary: "check pr"}, PhaseDraft, nil); err == nil || !strings.Contains(err.Error(), "no pull request link on a host gh is logged in to (github.com)") {
		t.Errorf("no link: err = %v", err)
	}
	note := &model.Note{ID: "n", Summary: "check pr https://github.com/acme/api/pull/1"}
	if _, err := Execute(context.Background(), spec, t.TempDir(), note, PhaseDraft, nil); err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("all failed: err = %v", err)
	}
}

func TestSummariseDraftSendsAtMostTenPullRequests(t *testing.T) {
	fetched := fakeFetchPRContext(t, nil)
	var links []string
	for number := 1; number <= 12; number++ {
		links = append(links, "https://github.com/acme/api/pull/"+strconv.Itoa(number))
	}
	note := &model.Note{ID: "n", Summary: "check pr", Body: strings.Join(links, "\n")}
	if _, err := Execute(context.Background(), summariseSpec(`echo '{"summary":"x"}'`), t.TempDir(), note, PhaseDraft, nil); err != nil {
		t.Fatal(err)
	}
	if len(*fetched) != maxSummarisedPRs {
		t.Errorf("fetched %d pull requests, want %d", len(*fetched), maxSummarisedPRs)
	}
}

func TestSummariseCreateWritesThePRSummaryWithoutClaude(t *testing.T) {
	root := t.TempDir()
	claudeRan := filepath.Join(root, "claude-ran")
	cfg := &config.Config{DigestRoot: root, Automations: []config.AutomationSpec{summariseSpec(`touch '` + claudeRan + `'`)}}
	noteStore := store.New(cfg.NotesDir())
	note := &model.Note{Summary: "check pr", Body: "https://github.com/acme/api/pull/1\n\n## PR summary\n\nstale", Status: model.StatusActive, Source: model.SourceManual, Created: time.Now()}
	if err := noteStore.Save(note); err != nil {
		t.Fatal(err)
	}
	if err := SaveDraft(cfg.AutomationDir(), note.ID, "summary: acme/api#1\ndescription: |-\n  ### acme/api#1 Add retries\n\n  Retries uploads.\n"); err != nil {
		t.Fatal(err)
	}
	if err := RunJob(context.Background(), cfg, note.ID, "summarise", PhaseCreate); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(claudeRan); !os.IsNotExist(err) {
		t.Error("create should not run claude for summarise")
	}
	saved, err := store.Load(note.FilePath)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Body != "https://github.com/acme/api/pull/1\n\n## PR summary\n\n### acme/api#1 Add retries\n\nRetries uploads." {
		t.Errorf("body =\n%s", saved.Body)
	}
	if saved.Automated != "" {
		t.Errorf("automated = %q, want unset so other automations stay offered", saved.Automated)
	}
	if _, err := LoadDraft(cfg.AutomationDir(), note.ID); !os.IsNotExist(err) {
		t.Errorf("draft kept after success: %v", err)
	}
	if Status(cfg.AutomationDir(), note.ID).HasDraft {
		t.Error("status still reports a draft")
	}
}

func TestSummariseRefusesBothPhasesWhenShowGitIsOff(t *testing.T) {
	root := t.TempDir()
	showGit := false
	cfg := &config.Config{DigestRoot: root, ShowGit: &showGit, Automations: []config.AutomationSpec{summariseSpec("true")}}
	note := &model.Note{Summary: "check pr", Body: "https://github.com/acme/api/pull/1", Status: model.StatusActive, Source: model.SourceManual, Created: time.Now()}
	if err := store.New(cfg.NotesDir()).Save(note); err != nil {
		t.Fatal(err)
	}
	ghCalls := fakeGHHosts(t, []string{"github.com"}, nil)
	for _, phase := range []Phase{PhaseDraft, PhaseCreate} {
		err := RunJob(context.Background(), cfg, note.ID, "summarise", phase)
		if err == nil || !strings.Contains(err.Error(), "summarise needs show_git: true") {
			t.Errorf("%s err = %v", phase, err)
		}
	}
	if *ghCalls != 0 {
		t.Errorf("gh ran %d times with show_git off", *ghCalls)
	}
}
