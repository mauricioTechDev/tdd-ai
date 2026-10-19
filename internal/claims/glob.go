package claims

import (
	"path"
	"sort"
	"strings"
)

// Match reports whether a slash-separated path matches a glob. "*" and "?"
// match within one segment (as in path.Match) and "**" matches any number of
// segments, including none.
func Match(pattern, name string) bool {
	return matchSegments(strings.Split(pattern, "/"), strings.Split(name, "/"))
}

func matchSegments(pattern, name []string) bool {
	for len(pattern) > 0 {
		if pattern[0] == "**" {
			for i := 0; i <= len(name); i++ {
				if matchSegments(pattern[1:], name[i:]) {
					return true
				}
			}
			return false
		}
		if len(name) == 0 {
			return false
		}
		if ok, err := path.Match(pattern[0], name[0]); err != nil || !ok {
			return false
		}
		pattern, name = pattern[1:], name[1:]
	}
	return len(name) == 0
}

// MatchAny reports whether name matches any of the patterns.
func MatchAny(patterns []string, name string) bool {
	for _, p := range patterns {
		if Match(p, name) {
			return true
		}
	}
	return false
}

// Generalize turns observed project paths into globs a check can hold steady:
// a file inside a directory becomes "dir/**" and a file at the root stays as
// it is. Globs already covered by a broader one are dropped.
func Generalize(paths []string) []string {
	set := map[string]bool{}
	for _, p := range paths {
		if dir := path.Dir(p); dir != "." {
			set[dir+"/**"] = true
		} else {
			set[p] = true
		}
	}
	out := []string{}
	for g := range set {
		covered := false
		for other := range set {
			if other != g && strings.HasSuffix(other, "/**") && Match(other, strings.TrimSuffix(g, "/**")) {
				covered = true
				break
			}
		}
		if !covered {
			out = append(out, g)
		}
	}
	sort.Strings(out)
	return out
}
