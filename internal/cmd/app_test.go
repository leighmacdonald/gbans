//nolint:testpackage // exercises the unexported log-address validation helper.
package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidLogAddressTarget(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"0.0.0.0:27715", false},
		{":27715", false},
		{"[::]:27715", false},
		{"127.0.0.1:27715", true},
		{"100.69.69.16:27715", true},
		{"gbans.example.com:27715", true},
		{"not-an-address", false},
	}

	for _, test := range tests {
		require.Equal(t, test.want, validLogAddressTarget(test.addr), test.addr)
	}
}
