package redact

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const placeholderPrefix = "[REDACTED:"

type Rule struct {
	Kind      string
	Pattern   *regexp.Regexp
	KeepGroup int
}

var BuiltInRules = []Rule{
	{Kind: "private_key", Pattern: regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`)},
	{Kind: "aws_access_key", Pattern: regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`)},
	{Kind: "github_token", Pattern: regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{36,}\b|\bgithub_pat_[A-Za-z0-9_]{22,}\b`)},
	{Kind: "slack_token", Pattern: regexp.MustCompile(`\bxox[abprs]-[A-Za-z0-9-]{10,}\b`)},
	{Kind: "api_key", Pattern: regexp.MustCompile(`\bsk-(?:ant-)?[A-Za-z0-9_-]{20,}\b`)},
	{Kind: "jwt", Pattern: regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)},
	{Kind: "bearer", Pattern: regexp.MustCompile(`(?i)\bbearer\s+([A-Za-z0-9._~+/=-]{16,})`), KeepGroup: 1},
	{Kind: "credential", Pattern: regexp.MustCompile(`(?i)\b(?:password|passwd|pwd|secret|token|api[_-]?key|auth[_-]?token|client[_-]?secret)\s*[:=]\s*["']?([^\s"']{6,})`), KeepGroup: 1},
	{Kind: "url_password", Pattern: regexp.MustCompile(`://[^/\s:@]+:([^/\s@]+)@`), KeepGroup: 1},
}

type Redactor struct {
	rules []Rule
}

type Report map[string]int

func New(extraPatterns []string) (*Redactor, error) {
	rules := slices.Clone(BuiltInRules)
	for _, expression := range extraPatterns {
		pattern, err := regexp.Compile(expression)
		if err != nil {
			return nil, fmt.Errorf("redact_patterns %q: %w", expression, err)
		}
		rules = append(rules, Rule{Kind: "custom", Pattern: pattern})
	}
	return &Redactor{rules: rules}, nil
}

func (r *Redactor) Apply(text string) (string, Report) {
	report := Report{}
	if r == nil {
		return text, report
	}
	for _, rule := range r.rules {
		text = replaceMatches(rule, text, report)
	}
	return text, report
}

func replaceMatches(rule Rule, text string, report Report) string {
	matches := rule.Pattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text
	}
	var redacted strings.Builder
	copiedUpTo := 0
	for _, match := range matches {
		start, end := match[2*rule.KeepGroup], match[2*rule.KeepGroup+1]
		if start < 0 || strings.HasPrefix(text[start:end], placeholderPrefix) {
			continue
		}
		redacted.WriteString(text[copiedUpTo:start])
		redacted.WriteString(placeholderPrefix)
		redacted.WriteString(rule.Kind)
		redacted.WriteString("]")
		copiedUpTo = end
		report[rule.Kind]++
	}
	redacted.WriteString(text[copiedUpTo:])
	return redacted.String()
}

func (report Report) Total() int {
	total := 0
	for _, count := range report {
		total += count
	}
	return total
}

func (report Report) String() string {
	total := report.Total()
	if total == 0 {
		return ""
	}
	kinds := make([]string, 0, len(report))
	for kind, count := range report {
		if count > 1 {
			kind += "×" + strconv.Itoa(count)
		}
		kinds = append(kinds, kind)
	}
	slices.Sort(kinds)
	noun := "secrets"
	if total == 1 {
		noun = "secret"
	}
	return fmt.Sprintf("redacted %d %s (%s)", total, noun, strings.Join(kinds, ", "))
}
