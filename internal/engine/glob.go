package engine

import "strings"

const Wildcards = "%*"

type Pattern struct {
	raw         string
	exact       string
	segments    []string
	isExact     bool
	matchAll    bool
	hasLeading  bool
	hasTrailing bool
}

func CompilePattern(pattern string) Pattern {
	p := Pattern{raw: pattern}

	if !strings.ContainsAny(pattern, Wildcards) {
		p.isExact = true
		p.exact = pattern
		return p
	}

	p.hasLeading = isWildcard(pattern[0])
	p.hasTrailing = isWildcard(pattern[len(pattern)-1])

	segments := make([]string, 0, strings.Count(pattern, "%")+strings.Count(pattern, "*")+1)
	start := -1
	for i := range len(pattern) {
		if isWildcard(pattern[i]) {
			if start >= 0 {
				segments = append(segments, pattern[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		segments = append(segments, pattern[start:])
	}

	p.segments = segments
	p.matchAll = len(segments) == 0
	return p
}

func (p Pattern) Match(name string) bool {
	if p.isExact {
		return name == p.exact
	}
	if p.matchAll {
		return true
	}

	segments := p.segments
	rest := name

	if !p.hasLeading && len(segments) > 0 {
		if !strings.HasPrefix(rest, segments[0]) {
			return false
		}
		rest = rest[len(segments[0]):]
		segments = segments[1:]
	}

	if !p.hasTrailing && len(segments) > 0 {
		last := segments[len(segments)-1]
		if !strings.HasSuffix(rest, last) {
			return false
		}
		rest = rest[:len(rest)-len(last)]
		segments = segments[:len(segments)-1]
	}

	for _, segment := range segments {
		at := strings.Index(rest, segment)
		if at < 0 {
			return false
		}
		rest = rest[at+len(segment):]
	}

	return true
}

func (p Pattern) String() string {
	return p.raw
}

func (p Pattern) IsExact() bool {
	return p.isExact
}

func isWildcard(c byte) bool {
	return c == '%' || c == '*'
}
