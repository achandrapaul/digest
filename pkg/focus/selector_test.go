package focus

import "testing"

func TestParseKeepsColonsInTheTitle(t *testing.T) {
	selector, err := Parse("acme:web-app:fix: keep the header: sticky")
	want := Selector{Org: "acme", Repo: "web-app", Title: "fix: keep the header: sticky"}
	if err != nil || selector != want {
		t.Fatalf("Parse = %+v, %v", selector, err)
	}
	if selector.String() != "acme:web-app:fix: keep the header: sticky" {
		t.Errorf("String = %q", selector.String())
	}
}

func TestParseRejectsIncompleteSelectors(t *testing.T) {
	for _, raw := range []string{"", "org", "org:repo", "org:repo:", ":repo:title", "org::title"} {
		if _, err := Parse(raw); err == nil {
			t.Errorf("Parse(%q) should fail", raw)
		}
	}
}
