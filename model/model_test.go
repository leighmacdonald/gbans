package model

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewBanNet(t *testing.T) {
	_, err := NewBanNet("172.16.1.0/24", "test", time.Minute*10, System)
	require.NoError(t, err)
}
