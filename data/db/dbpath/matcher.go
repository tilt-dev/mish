// Helpers for matching db paths.
package dbpath

import (
	"encoding/json"
	"path"
	"strings"

	"github.com/windmilleng/mish/data/pathutil"
)

type Matcher struct {
	pathutil.Matcher
}

func (m *Matcher) Equal(m2 *Matcher) bool {
	ps := m.ToPatterns()
	ps2 := m2.ToPatterns()
	if len(ps) != len(ps2) {
		return false
	}

	for i, p := range ps {
		p2 := ps2[i]
		if p != p2 {
			return false
		}
	}
	return true
}

// Sadly, golang doesn't support covariant return types. :(
func (m *Matcher) SubdirDB(path string) *Matcher {
	return &Matcher{Matcher: m.Matcher.Subdir(path)}
}

func (m *Matcher) ChildDB(path string) *Matcher {
	return &Matcher{Matcher: m.Matcher.Child(path)}
}

func (m *Matcher) Invert() (*Matcher, error) {
	inverted, err := pathutil.InvertMatcher(m.Matcher)
	if err != nil {
		return nil, err
	}

	return &Matcher{Matcher: inverted}, nil
}

func (m *Matcher) Subdir(path string) pathutil.Matcher {
	return m.SubdirDB(path)
}

func (m *Matcher) Child(path string) pathutil.Matcher {
	return m.ChildDB(path)
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

type dbPathUtil struct{}

func (d dbPathUtil) Base(p string) string                  { return path.Base(p) }
func (d dbPathUtil) Dir(path string) string                { return Dir(path) }
func (d dbPathUtil) Join(a, b string) string               { return path.Join(a, b) }
func (d dbPathUtil) Match(pattern, p string) (bool, error) { return path.Match(pattern, p) }
func (d dbPathUtil) Separator() rune                       { return '/' }

var theDbPathUtil = dbPathUtil{}

// Matches nothing.
func NewEmptyMatcher() *Matcher {
	return &Matcher{Matcher: pathutil.NewEmptyMatcher()}
}

// Matches everything
func NewAllMatcher() *Matcher {
	return &Matcher{Matcher: pathutil.NewAllMatcher()}
}

// Matches a single file only
func NewFileMatcher(file string) (*Matcher, error) {
	m, err := pathutil.NewFileMatcher(theDbPathUtil, file)
	return &Matcher{Matcher: m}, err
}

func NewFileMatcherOrPanic(file string) *Matcher {
	m, err := NewFileMatcher(file)
	if err != nil {
		panic(err)
	}
	return m
}

func NewFilesMatcher(files []string) (*Matcher, error) {
	m, err := pathutil.NewFilesMatcher(theDbPathUtil, files)
	return &Matcher{Matcher: m}, err
}

func NewMatcherFromPattern(pattern string) (*Matcher, error) {
	m, err := pathutil.NewMatcherFromPattern(theDbPathUtil, pattern)
	return &Matcher{Matcher: m}, err
}

func NewMatcherFromPatterns(patterns []string) (*Matcher, error) {
	m, err := pathutil.NewMatcherFromPatterns(theDbPathUtil, patterns)
	return &Matcher{Matcher: m}, err
}

func ToString(m *Matcher) string {
	return strings.Join(m.ToPatterns(), ",")
}

func MatchersEqual(a, b *Matcher) bool {
	return pathutil.MatchersEqual(a.Matcher, b.Matcher)
}
