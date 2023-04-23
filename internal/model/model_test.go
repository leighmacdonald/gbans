package model

import (
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFilter_Match(t *testing.T) {
	filter := Filter{
		FilterID:  1,
		Pattern:   `^poo`,
		IsRegex:   true,
		CreatedOn: config.Now(),
	}
	filter.Init()
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
