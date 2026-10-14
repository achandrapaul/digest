package focus

import (
	"fmt"
	"strings"
)

type Selector struct {
	Org   string `json:"org"`
	Repo  string `json:"repo"`
	Title string `json:"title"`
}

func Parse(raw string) (Selector, error) {
	parts := strings.SplitN(raw, ":", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || strings.TrimSpace(parts[2]) == "" {
		return Selector{}, fmt.Errorf("want <org>:<repo>:<pr title>, got %q", raw)
	}
	return Selector{Org: parts[0], Repo: parts[1], Title: parts[2]}, nil
}

func (s Selector) String() string {
	return s.Org + ":" + s.Repo + ":" + s.Title
}
