package bantype_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/stretchr/testify/require"
)

func TestType_String(t *testing.T) {
	tests := []struct {
		name string
		typ  bantype.Type
		want string
	}{
		{name: "OK", typ: bantype.OK, want: ""},
		{name: "NoComm", typ: bantype.NoComm, want: "mute/gag"},
		{name: "Banned", typ: bantype.Banned, want: "banned"},
		{name: "Network", typ: bantype.Network, want: "network"},
		{name: "Unknown", typ: bantype.Unknown, want: "unknown"},
		{name: "out of range positive", typ: bantype.Type(99), want: ""},
		{name: "out of range negative", typ: bantype.Type(-5), want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, tt.typ.String())
		})
	}
}
