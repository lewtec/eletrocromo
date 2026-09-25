package notification_test

import (
	"testing"

	"github.com/lewtec/eletrocromo/driver/notification"
	_ "github.com/lewtec/eletrocromo/driver/notification/discard"
	"github.com/stretchr/testify/require"
)

func TestNotify_Discard(t *testing.T) {
	err := notification.Notify(t.Context(), notification.Notification{
		Title:   "hi",
		Message: "there",
	})
	require.NoError(t, err)
}
