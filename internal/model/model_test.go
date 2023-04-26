package model

import (
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/stretchr/testify/require"
	"strings"
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
	matched := false
	for _, word := range strings.Split("super pooooooper", " ") {
		if filter.Match(word) {
			matched = true
		}
	}
	require.True(t, matched)
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
