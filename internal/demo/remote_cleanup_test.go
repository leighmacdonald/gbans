//nolint:testpackage // exercises the unexported remote cleanup decision directly.
package demo

import (
	"errors"
	"fmt"
	"testing"

	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/stats"
	"github.com/stretchr/testify/assert"
)

func TestShouldRemoveRemoteDemo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "too large",
			err:  asset.ErrAssetTooLarge,
			want: true,
		},
		{
			name: "not enough players",
			err:  fmt.Errorf("%w: %w", stats.ErrMatchFiltered, stats.ErrNotEnoughPlayers),
			want: true,
		},
		{
			name: "joined cleanup error",
			err:  errors.Join(asset.ErrDeleteAssetFile, fmt.Errorf("%w: %w", stats.ErrMatchFiltered, stats.ErrNotEnoughPlayers)),
			want: true,
		},
		{
			name: "transient error",
			err:  ErrParse,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.want, shouldRemoveRemoteDemo(test.err))
		})
	}
}
