// Helpers for matching file paths.
package ospath

import (
	"encoding/json"
	"strings"

	"github.com/windmilleng/mish/data/pathutil"
)

type Matcher struct {
	pathutil.Matcher
}

func (m *Matcher) SubdirOS(path string) *Matcher {
	return &Matcher{Matcher: m.Matcher.Subdir(path)}
}

func (m *Matcher) ChildOS(path string) *Matcher {
	return &Matcher{Matcher: m.Matcher.Child(path)}
}

func (m *Matcher) Subdir(path string) pathutil.Matcher {
	return m.SubdirOS(path)
}

func (m *Matcher) Child(path string) pathutil.Matcher {
	return m.ChildOS(path)
}

func (m *Matcher) MarshalJSON() ([]byte, error) {
	patterns := m.Matcher.ToPatterns()
	return json.Marshal(patterns)
}

func (m *Matcher) UnmarshalJSON(b []byte) error {
	var patterns []string
	err := json.Unmarshal(b, &patterns)
	if err != nil {
		return err
	}

	newM, err := NewMatcherFromPatterns(patterns)
	if err != nil {
		return err
	}
	m.Matcher = newM.Matcher
	return nil
}

// Matches nothing.
func NewEmptyMatcher() *Matcher {
	return &Matcher{Matcher: pathutil.NewEmptyMatcher()}
}

// Matches everything
func NewAllMatcher() *Matcher {
	return &Matcher{Matcher: pathutil.NewAllMatcher()}
}

func InvertMatcher(m *Matcher) (*Matcher, error) {
	inner, err := pathutil.InvertMatcher(m.Matcher)
	if err != nil {
		return &Matcher{}, err
	}
	return &Matcher{Matcher: inner}, nil
}

// Matches a single file only
func NewFileMatcher(file string) (*Matcher, error) {
	m, err := pathutil.NewFileMatcher(theOsPathUtil, file)
	return &Matcher{Matcher: m}, err
}

func NewMatcherFromPattern(pattern string) (*Matcher, error) {
	m, err := pathutil.NewMatcherFromPattern(theOsPathUtil, pattern)
	return &Matcher{Matcher: m}, err
}

func NewMatcherFromPatterns(patterns []string) (*Matcher, error) {
	m, err := pathutil.NewMatcherFromPatterns(theOsPathUtil, patterns)
	return &Matcher{Matcher: m}, err
}

func (m *Matcher) String() string {
	return strings.Join(m.ToPatterns(), ",")
}

func MatchersEqual(a, b *Matcher) bool {
	return pathutil.MatchersEqual(a.Matcher, b.Matcher)
}
