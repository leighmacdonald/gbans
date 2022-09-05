package app

import (
	"github.com/leighmacdonald/gbans/internal/model"
	"sync"
)

var (
	wordFilters   []model.Filter
	wordFiltersMu *sync.RWMutex
)

func init() {
	wordFiltersMu = &sync.RWMutex{}
}

// importFilteredWords loads the supplied word list into memory
func importFilteredWords(filters []model.Filter) {
	wordFiltersMu.Lock()
	defer wordFiltersMu.Unlock()
	wordFilters = filters
}
