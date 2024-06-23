// Package util provides some useful functions that don't fit anywhere else.
package stringutil

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/sergi/go-diff/diffmatchpatch"
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

func SanitizeLog(s string) string {
	for _, char := range []string{"\n", "\r"} {
		s = strings.ReplaceAll(s, char, "")
	}

	return s
}

func DiffString(s1, s2 string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffPrettyText(dmp.DiffMain(s1, s2, true))

	return fmt.Sprintf("```diff\n%s```", diffs)
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

func ToLowerSlice(stringSlice []string) []string {
	lowerSlice := make([]string, len(stringSlice))

	for i := range stringSlice {
		lowerSlice[i] = strings.ToLower(stringSlice[i])
	}

	return lowerSlice
}
