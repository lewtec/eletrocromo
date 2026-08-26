package notification_test

import (
	"testing"

	"github.com/lewtec/eletrocromo/driver/notification"
	_ "github.com/lewtec/eletrocromo/driver/notification/discard"
)

func TestNotify_Discard(t *testing.T) {
	err := notification.Notify(t.Context(), notification.Notification{
		Title:   "hi",
		Message: "there",
	})
	if err != nil {
		t.Fatal(err)
	}
}
