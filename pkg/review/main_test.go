package review

import (
	"os"
	"testing"

	"github.com/achandrapaul/digest/pkg/notify"
)

func TestMain(m *testing.M) {
	sendNotification = func(string, string, string) error { return nil }
	sendReviewNotification = func(notify.Notification) error { return nil }
	os.Exit(m.Run())
}
