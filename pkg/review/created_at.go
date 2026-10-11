package review

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/achandrapaul/digest/pkg/system"
)

var runGHPRView = func(ctx context.Context, url string) ([]byte, error) {
	return system.GHCommand(ctx, "pr", "view", url, "--json", "createdAt", "--jq", ".createdAt").Output()
}

func FetchPRCreatedAt(ctx context.Context, url string) (time.Time, error) {
	output, err := runGHPRView(ctx, url)
	if err != nil {
		return time.Time{}, fmt.Errorf("gh pr view %s: %w", url, err)
	}
	return time.Parse(time.RFC3339, strings.TrimSpace(string(output)))
}
