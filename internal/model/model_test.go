package model

import (
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFilter_Match(t *testing.T) {
	filter := Filter{
		WordID:    1,
		Patterns:  []string{`*poo*oop*`},
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
