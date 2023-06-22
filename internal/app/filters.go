package app

import (
	"sync"

	"github.com/leighmacdonald/gbans/internal/store"
)

var (
	wordFilters   []store.Filter
	wordFiltersMu *sync.RWMutex
)

func init() {
	wordFiltersMu = &sync.RWMutex{}
}

// importFilteredWords loads the supplied word list into memory
func importFilteredWords(filters []store.Filter) {
	wordFiltersMu.Lock()
	defer wordFiltersMu.Unlock()
	wordFilters = filters
}
