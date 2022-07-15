package model

import (
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

func TestNewBanNet(t *testing.T) {
	_, errBanNet := NewBanNet("172.16.1.0/24", Language, time.Minute*10, System)
	require.NoError(t, errBanNet)
}

func TestFilter_Match(t *testing.T) {
	filter := Filter{
		WordID:    1,
		Pattern:   regexp.MustCompile(`(po+p)`),
		CreatedOn: config.Now(),
	}
	require.True(t, filter.Match("super pooooooper"))
}

//
//func TestServerEvent(t *testing.T) {
//	se := ServerEvent{
//		MetaData: map[string]any{
//			"crit": "1",
//			"headshot": "0",
//		},
//	}
//}
