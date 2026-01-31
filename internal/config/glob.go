package config

import (
	"path"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// GlobMatcher matches branch names against glob patterns
type GlobMatcher struct {
	patterns []string
}

// NewGlobMatcher creates a new GlobMatcher with the given patterns
func NewGlobMatcher(patterns []string) *GlobMatcher {
	return &GlobMatcher{patterns: patterns}
}

// Matches returns true if the branch name matches any of the patterns
func (g *GlobMatcher) Matches(branchName string) bool {
	name := strings.TrimPrefix(branchName, "refs/heads/")
	name = strings.TrimPrefix(name, "refs/remotes/")

	for _, pattern := range g.patterns {
		if strings.Contains(pattern, "**") {
			matched, err := doublestar.Match(pattern, name)
			if err == nil && matched {
				return true
			}
			continue
		}

		matched, err := path.Match(pattern, name)
		if err == nil && matched {
			return true
		}

		parts := strings.Split(name, "/")
		lastPart := parts[len(parts)-1]
		matched, err = path.Match(pattern, lastPart)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// AddPattern adds a pattern to the matcher
func (g *GlobMatcher) AddPattern(pattern string) {
	g.patterns = append(g.patterns, pattern)
}

// Patterns returns the current patterns
func (g *GlobMatcher) Patterns() []string {
	return g.patterns
}

// IsEmpty returns true if there are no patterns
func (g *GlobMatcher) IsEmpty() bool {
	return len(g.patterns) == 0
}
