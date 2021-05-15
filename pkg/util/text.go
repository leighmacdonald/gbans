package util

import (
	"github.com/leighmacdonald/gbans/internal/model"
	"strings"
	"sync"
)

var (
	wordFilters   []*model.Filter
	wordFiltersMu *sync.RWMutex
)

// ImportFilteredWords loads the supplied word list into memory
func ImportFilteredWords(filters []*model.Filter) {
	wordFiltersMu.Lock()
	defer wordFiltersMu.Unlock()
	wordFilters = filters
}

// IsFilteredWord checks to see if the body of text contains a known filtered word
func IsFilteredWord(body string) (bool, *model.Filter) {
	if body == "" {
		return false, nil
	}
	wordFiltersMu.RLock()
	defer wordFiltersMu.RUnlock()
	ls := strings.ToLower(body)
	for _, filter := range wordFilters {
		if filter.Match(ls) {
			return true, filter
		}
	}
	return false, nil
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
		curLineSize := len(s) + len(sepChar) // account for \n
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
	wordFiltersMu = &sync.RWMutex{}
}
