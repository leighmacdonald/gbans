// Package stringutil provides some string based helpers.
package stringutil

import (
	"crypto/rand"
	"math/big"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

// StringChunkDelimited is used to split a multiline string into strings with a max size defined as chunkSize.
// A string of len > chunkSize will not be split.
func StringChunkDelimited(data string, chunkSize int, sep ...string) []string {
	if len(data) <= chunkSize {
		return []string{data}
	}

	var ( //nolint:prealloc
		results   []string
		curPieces []string
		curSize   int
		sepChar   = "\n"
	)

	if len(sep) > 0 {
		sepChar = sep[0]
	}

	rows := strings.Split(data, sepChar)
	for index, row := range rows {
		curLineSize := len(row) + len(sepChar) // account for \n
		if curSize+curLineSize >= chunkSize {
			results = append(results, strings.TrimSuffix(strings.Join(curPieces, sepChar), sepChar))
			curSize = 0
			curPieces = nil
		}

		curPieces = append(curPieces, row)
		curSize += curLineSize

		if index+1 == len(rows) {
			results = append(results, strings.TrimSuffix(strings.Join(curPieces, sepChar), sepChar))
		}
	}

	return results
}

func SanitizeUGC(body string) string {
	return bluemonday.UGCPolicy().Sanitize(body)
}

func SecureRandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-"

	ret := make([]byte, n)

	for currentChar := range n {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return ""
		}

		ret[currentChar] = letters[num.Int64()]
	}

	return string(ret)
}
