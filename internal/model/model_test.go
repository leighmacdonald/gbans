package model

import (
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

func TestNewBanNet(t *testing.T) {
	_, err := NewBanNet("172.16.1.0/24", "test", time.Minute*10, System)
	require.NoError(t, err)
}

func TestFilter_Match(t *testing.T) {
	f := Filter{
		WordID:    1,
		Word:      regexp.MustCompile(`(po+p)`),
		CreatedOn: time.Now(),
	}
	require.True(t, f.Match("super pooooooper"))
}
