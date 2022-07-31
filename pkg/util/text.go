// Package util provides some useful functions that don't fit anywhere else.
package util

import (
	"fmt"
	"github.com/sergi/go-diff/diffmatchpatch"
	"strings"
)

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

func SanitizeLog(s string) string {
	for _, char := range []string{"\n", "\r"} {
		s = strings.Replace(s, char, "", -1)
	}
	return s
}

func DiffString(s1, s2 string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffPrettyText(dmp.DiffMain(s1, s2, true))
	return fmt.Sprintf("```diff\n%s```", diffs)
}

const GLOB = "*"

func GlobString(pattern, subj string) bool {
	if pattern == "" {
		return subj == pattern
	}
	if pattern == GLOB {
		return true
	}
	parts := strings.Split(pattern, GLOB)
	if len(parts) == 1 {
		return subj == pattern
	}
	leadingGlob := strings.HasPrefix(pattern, GLOB)
	trailingGlob := strings.HasSuffix(pattern, GLOB)
	end := len(parts) - 1
	for i := 0; i < end; i++ {
		idx := strings.Index(subj, parts[i])

		switch i {
		case 0:
			if !leadingGlob && idx != 0 {
				return false
			}
		default:
			if idx < 0 {
				return false
			}
		}
		subj = subj[idx+len(parts[i]):]
	}
	return trailingGlob || strings.HasSuffix(subj, parts[end])
}
