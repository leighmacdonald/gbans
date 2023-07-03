package app

import (
	"strings"
	"sync"

	"github.com/leighmacdonald/gbans/internal/store"
)

type wordFilters struct {
	*sync.RWMutex
	wordFilters []store.Filter
}

func newWordFilters() *wordFilters {
	return &wordFilters{
		RWMutex: &sync.RWMutex{},
	}
}

// importFilteredWords loads the supplied word list into memory.
func (f *wordFilters) importFilteredWords(filters []store.Filter) {
	f.Lock()
	defer f.Unlock()
	f.wordFilters = filters
}

// findFilteredWordMatch checks to see if the body of text contains a known filtered word
// It will only return the first matched filter found.
func (f *wordFilters) findFilteredWordMatch(body string) (string, *store.Filter) {
	if body == "" {
		return "", nil
	}

	words := strings.Split(strings.ToLower(body), " ")

	f.RLock()
	defer f.RUnlock()

	for _, filter := range f.wordFilters {
		for _, word := range words {
			if filter.Match(word) {
				return word, &filter
			}
		}
	}

	return "", nil
}
