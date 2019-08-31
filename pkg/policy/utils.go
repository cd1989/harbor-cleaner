package policy

import "path"

// Check whether to retain the tag against the patterns.
func Retain(patterns []string, tag string) bool {
	for _, pattern := range patterns {
		m, e := path.Match(pattern, tag)
		if e == nil && m {
			return true
		}
	}

	return false
}
