package util

import (
	"strings"
	"sync"
)

var (
	filteredWords   []string
	filteredWordsMu *sync.RWMutex
)

// ImportFilteredWords loads the supplied word list into memory
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

// IsFilteredWord checks to see if the body of text contains a known filtered word
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

// StringChunkDelimited is used to split a multiline string into strings with a max size defined as chunkSize.
// A string of len > chunkSize will not be split.
func StringChunkDelimited(data string, chunkSize int, sep ...string) []string {
	if len(data) <= chunkSize {
		return []string{data}
	}
	var results []string
	var curPieces []string
	var curSize int
	sepChar := "\n"
	if len(sep) > 0 {
		sepChar = sep[0]
	}
	rows := strings.Split(data, sepChar)
	for i, s := range rows {
		curLineSize := len(s) + 1 // account for \n
		if curSize+curLineSize >= chunkSize {
			results = append(results, strings.TrimSuffix(strings.Join(curPieces, sepChar), sepChar))
			curSize = 0
			curPieces = nil
		}
		curPieces = append(curPieces, s)
		curSize += curLineSize
		if i+1 == len(rows) {
			results = append(results, strings.TrimSuffix(strings.Join(curPieces, sepChar), sepChar))
		}
	}
	return results
}

func init() {
	filteredWordsMu = &sync.RWMutex{}
}
