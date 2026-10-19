package opengraph

import (
	"strings"
)

type segKind int

const (
	segRoot segKind = iota
	segLiteral
	segWildcard
	segCatchAll
)

type segment struct {
	kind    segKind
	name    string
	literal string
}

// compilePattern parses a route pattern into matchable segments. It accepts the
// same wildcard syntax as the http.ServeMux patterns used by the frontend
// routes: single segment wildcards ({name}) and catch all wildcards
// ({name...}), only valid as the final segment.
func compilePattern(pattern string) ([]segment, error) {
	if pattern == "/" {
		return []segment{{kind: segRoot}}, nil
	}

	if len(pattern) < 2 || !strings.HasPrefix(pattern, "/") || strings.HasSuffix(pattern, "/") {
		return nil, ErrInvalidPattern
	}

	parts := strings.Split(strings.TrimPrefix(pattern, "/"), "/")
	segments := make([]segment, 0, len(parts))

	for partIndex, part := range parts {
		if part == "" {
			return nil, ErrInvalidPattern
		}

		if !strings.HasPrefix(part, "{") || !strings.HasSuffix(part, "}") {
			segments = append(segments, segment{kind: segLiteral, literal: part})

			continue
		}

		name := part[1 : len(part)-1]

		if strings.HasSuffix(name, "...") {
			// Catch all wildcards must be the final segment.
			if partIndex != len(parts)-1 {
				return nil, ErrInvalidPattern
			}

			name = strings.TrimSuffix(name, "...")

			if name == "" {
				return nil, ErrInvalidPattern
			}

			segments = append(segments, segment{kind: segCatchAll, name: name})

			continue
		}

		if name == "" {
			return nil, ErrInvalidPattern
		}

		segments = append(segments, segment{kind: segWildcard, name: name})
	}

	return segments, nil
}

// matchSegments matches a compiled pattern against the split request path.
func matchSegments(pattern []segment, parts []string) (map[string]string, bool) {
	params := make(map[string]string, len(pattern))

	if len(pattern) == 1 && pattern[0].kind == segRoot {
		return nil, len(parts) == 1 && parts[0] == ""
	}

	catchAll := len(pattern) > 0 && pattern[len(pattern)-1].kind == segCatchAll
	if !catchAll && len(pattern) != len(parts) {
		return nil, false
	}

	if catchAll && len(parts) < len(pattern) {
		return nil, false
	}

	for idx, seg := range pattern {
		part := parts[idx]

		switch seg.kind {
		case segCatchAll:
			params[seg.name] = strings.Join(parts[idx:], "/")
		case segWildcard:
			if part == "" {
				return nil, false
			}

			params[seg.name] = part
		case segLiteral:
			if seg.literal != part {
				return nil, false
			}
		case segRoot:
			return nil, false
		}
	}

	return params, true
}
