package reason_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/stretchr/testify/require"
)

func TestReason_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		reason reason.Reason
		want   string
	}{
		{name: "Custom", reason: reason.Custom, want: "Custom"},
		{name: "External", reason: reason.External, want: "3rd party"},
		{name: "Cheating", reason: reason.Cheating, want: "Cheating"},
		{name: "Racism", reason: reason.Racism, want: "Racism"},
		{name: "Harassment", reason: reason.Harassment, want: "Personal Harassment"},
		{name: "Exploiting", reason: reason.Exploiting, want: "Exploiting"},
		{name: "WarningsExceeded", reason: reason.WarningsExceeded, want: "Warnings Exceeded"},
		{name: "Spam", reason: reason.Spam, want: "Spam"},
		{name: "Language", reason: reason.Language, want: "Language"},
		{name: "Profile", reason: reason.Profile, want: "Profile"},
		{name: "ItemDescriptions", reason: reason.ItemDescriptions, want: "Item Name or Descriptions"},
		{name: "BotHost", reason: reason.BotHost, want: "BotHost"},
		{name: "Evading", reason: reason.Evading, want: "Evading"},
		{name: "Username", reason: reason.Username, want: "Inappropriate Username"},
		{name: "zero value is empty", reason: 0, want: ""},
		{name: "out of range", reason: 99, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, tt.reason.String())
		})
	}
}
