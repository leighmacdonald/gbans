package util

import (
	"strings"
	"sync"
)

var (
	filteredWords []string
	filteredWordsMu *sync.RWMutex
)

func ImportFilteredWords(words []string) {
	var contains = func(lWord string) bool {
		for _, w := range filteredWords {
			if lWord == w {
				return true
			}
		}
		return false
	}
	for _, fWord := range words {
		if !contains(strings.ToLower(fWord)) {
			filteredWordsMu.Lock()
			filteredWords = append(filteredWords, strings.ToLower(fWord))
			filteredWordsMu.Unlock()
		}
	}
}

func IsFilteredWord(body string) (bool, string) {
	if body == "" {
		return false, ""
	}
	filteredWordsMu.RLock()
	defer filteredWordsMu.RUnlock()
	for _, word := range strings.Split(strings.ToLower(body), " ") {
		if word == "" {
			continue
		}
		for _, fWord := range filteredWords {
			if word == fWord {
				return true, word
			}
		}
	}
	return false, ""
}

func init() {
	filteredWordsMu = &sync.RWMutex{}
}