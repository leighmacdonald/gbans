package config_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/stretchr/testify/require"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"3s",
			"3s",
		},
		{
			"3m",
			"3m0s",
		},
		{
			"3h",
			"3h0m0s",
		},
		{
			"3d",
			"72h0m0s",
		},
		{
			"3w",
			"504h0m0s",
		},
		{
			"3M",
			"2232h0m0s",
		},
		{
			"3y",
			"26280h0m0s",
		},
	}
	for _, test := range tests {
		d, errParseDuration := config.ParseDuration(test.input)
		require.NoError(t, errParseDuration, "Failed to parse: %s", test.input)
		require.Equal(t, test.expected, d.String(), "Failed to parse: %s", test.input)
	}
}
