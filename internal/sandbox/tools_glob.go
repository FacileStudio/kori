package sandbox

import (
	"path/filepath"
	"strings"
)

func matchStar(pattern, name []string) bool {
	if len(pattern) == 0 {
		return true
	}
	for i := 0; i <= len(name); i++ {
		if matchSegments(pattern, name[i:], matchStar) {
			return true
		}
	}
	return false
}

func matchSegments(pattern, name []string, star func([]string, []string) bool) bool {
	p := pattern
	n := name
	for len(p) > 0 {
		if p[0] == "**" {
			return star(p[1:], n)
		}
		if len(n) == 0 {
			return false
		}
		if ok, err := filepath.Match(p[0], n[0]); err != nil || !ok {
			return false
		}
		p = p[1:]
		n = n[1:]
	}
	return len(n) == 0
}

func matchGlob(pattern, name string) bool {
	p := strings.Split(pattern, "/")
	n := strings.Split(name, "/")
	return matchSegments(p, n, matchStar)
}
