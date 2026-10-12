package redact_test

import (
	"strings"
	"testing"

	"github.com/achandrapaul/digest/pkg/redact"
)

func TestApplyRedactsEachBuiltInKind(t *testing.T) {
	privateKeyBlock := "-----BEGIN RSA PRIVATE KEY-----\nMIIEow" + strings.Repeat("A", 40) + "\n-----END RSA PRIVATE KEY-----"
	cases := []struct {
		kind, input, want string
	}{
		{"private_key", "key:\n" + privateKeyBlock + "\ndone", "key:\n[REDACTED:private_key]\ndone"},
		{"aws_access_key", "aws AKIA" + strings.Repeat("Q", 16) + " here", "aws [REDACTED:aws_access_key] here"},
		{"github_token", "gh ghp_" + strings.Repeat("a", 36), "gh [REDACTED:github_token]"},
		{"github_token", "pat github_pat_" + strings.Repeat("b", 30), "pat [REDACTED:github_token]"},
		{"slack_token", "slack xoxb-" + strings.Repeat("1", 12), "slack [REDACTED:slack_token]"},
		{"api_key", "llm sk-ant-" + strings.Repeat("z", 24), "llm [REDACTED:api_key]"},
		{"jwt", "jwt eyJhbGciOi.eyJzdWIiOi.c2lnbmF0dXJl", "jwt [REDACTED:jwt]"},
		{"bearer", "Authorization: Bearer " + strings.Repeat("t", 20), "Authorization: Bearer [REDACTED:bearer]"},
		{"credential", "password=hunter22", "password=[REDACTED:credential]"},
		{"credential", `api_key: "abcdef123"`, `api_key: "[REDACTED:credential]"`},
		{"url_password", "https://deploy:s3cretpass@example.com/x", "https://deploy:[REDACTED:url_password]@example.com/x"},
	}
	redactor, err := redact.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range cases {
		got, report := redactor.Apply(testCase.input)
		if got != testCase.want {
			t.Errorf("%s: got %q, want %q", testCase.kind, got, testCase.want)
		}
		if report[testCase.kind] != 1 {
			t.Errorf("%s: report %v, want one %s", testCase.kind, report, testCase.kind)
		}
	}
}

func TestApplyLeavesOrdinaryNotesAlone(t *testing.T) {
	redactor, err := redact.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	note := "Fix the token refresh bug in auth.go\nReset password flow ships Friday\nSee https://example.com/docs and PR #42"
	got, report := redactor.Apply(note)
	if got != note || report.Total() != 0 {
		t.Fatalf("got %q with report %v, want unchanged", got, report)
	}
}

func TestCustomPatternsRedactWholeMatch(t *testing.T) {
	redactor, err := redact.New([]string{`INTERNAL-[0-9]{6}`})
	if err != nil {
		t.Fatal(err)
	}
	got, report := redactor.Apply("code INTERNAL-123456 and INTERNAL-654321")
	if got != "code [REDACTED:custom] and [REDACTED:custom]" || report["custom"] != 2 {
		t.Fatalf("got %q with report %v", got, report)
	}
}

func TestNewRejectsInvalidPattern(t *testing.T) {
	if _, err := redact.New([]string{"("}); err == nil || !strings.Contains(err.Error(), "redact_patterns") {
		t.Fatalf("err = %v, want a redact_patterns error", err)
	}
}

func TestEarlierPlaceholderIsNotRedactedAgain(t *testing.T) {
	redactor, _ := redact.New(nil)
	got, report := redactor.Apply("token=ghp_" + strings.Repeat("a", 36))
	if got != "token=[REDACTED:github_token]" || report.Total() != 1 {
		t.Fatalf("got %q with report %v", got, report)
	}
}

func TestNilRedactorPassesTextThrough(t *testing.T) {
	var redactor *redact.Redactor
	if got, report := redactor.Apply("password=hunter22"); got != "password=hunter22" || report.Total() != 0 {
		t.Fatalf("got %q with report %v", got, report)
	}
}

func TestReportStringNeverIncludesValues(t *testing.T) {
	redactor, _ := redact.New(nil)
	_, report := redactor.Apply("password=hunter22 token=abcdef99 ghp_" + strings.Repeat("a", 36))
	summary := report.String()
	if summary != "redacted 3 secrets (credential×2, github_token)" {
		t.Fatalf("summary = %q", summary)
	}
	if strings.Contains(summary, "hunter22") || (redact.Report{}).String() != "" {
		t.Fatalf("summary leaked a value or empty report was not blank: %q", summary)
	}
}
