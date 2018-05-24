// Helpers for matching both DB paths and File paths.
package pathutil

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// A Matcher is a limited propositional logic engine for choosing a subset of files
// in a file tree.
//
// By design, we don't try to implement a logic engine that allows arbitrarily
// complex boolean formulas. For example, our current logic engine does not
// support the matcher
//
// (foo/** && ((NOT foo/bar/**) || foo/bar/baz/**))
//
// Right now, we try only to support matchers in a normal form:
//
// (A or B) and (not C) and (not D)
//
// or equivalently
//
// (A or B) and not (C or D)
//
// This is not super formal right now. One of the sad limitations of this engine
// is that there are cases where we can express a boolean formula but not its inverse.
// For example, (foo/** && (NOT foo/bar/**)) is expressible but its inverse is not.
type Matcher interface {
	Match(s string) bool
	ToPatterns() []string

	// True if are certain this Matcher won't match anything.
	Empty() bool

	// True if are certain this Matcher matches everything
	All() bool

	// Whether this is a well-formed Matcher. Verify that we only accept matchers written in normal form:
	// (A or B or C) and (not D) and (not E)
	IsNormal() bool

	// If this matcher will only match a discrete set of files, return the file path.
	AsFileSet() []string

	// Create a new matcher that matches prefix/{originalMatch}.
	// i.e., if m.Matches('a') is true, then m.Subdir('b').Matches('b/a') is true.
	Subdir(prefix string) Matcher

	// Create a new matcher that matches children of the original match pattern.
	// i.e., if m.Matches('b/a') is true, then m.Child('b').Matches('a') is true.
	Child(prefix string) Matcher
}

// Inverts a matcher
type invertMatcher struct {
	matcher Matcher
}

func InvertMatcher(m Matcher) (Matcher, error) {
	if m.Empty() {
		return NewAllMatcher(), nil
	} else if m.All() {
		return NewEmptyMatcher(), nil
	} else if listMatcher, ok := m.(listMatcher); ok {
		// DeMorgan's rule:
		// not (A or B) = not A and not B
		// not (A and B) = not A or not B
		// But not all inverted matchers can be written in normal form,
		// so we need to make sure that the result is normal.
		matchers := listMatcher.matchers
		inverted := make([]Matcher, len(matchers))
		for i, m := range matchers {
			im, err := InvertMatcher(m)
			if err != nil {
				return nil, err
			}
			inverted[i] = im
		}
		result := newListMatcher(!listMatcher.conjunction, inverted)
		if !result.IsNormal() {
			return nil, fmt.Errorf("Inverted matcher cannot be written in normal form: %v", m.ToPatterns())
		}
		return result, nil
	} else if invertedMatcher, ok := m.(invertMatcher); ok {
		return invertedMatcher.matcher, nil
	}
	return invertMatcher{matcher: m}, nil
}

func (m invertMatcher) ToPatterns() []string {
	patterns := m.matcher.ToPatterns()
	for i, p := range patterns {
		if isInverted(p) {
			patterns[i] = p[1:]
		} else {
			patterns[i] = "!" + p
		}
	}
	return patterns
}

func (m invertMatcher) Match(path string) bool { return !m.matcher.Match(path) }
func (m invertMatcher) Empty() bool            { return m.matcher.Empty() }
func (m invertMatcher) All() bool              { return m.matcher.All() }
func (m invertMatcher) AsFileSet() []string    { return nil }

func (m invertMatcher) IsNormal() bool {
	_, isList := m.matcher.(listMatcher)
	return !isList && m.matcher.IsNormal()
}

func (m invertMatcher) Subdir(prefix string) Matcher {
	i, err := InvertMatcher(m.matcher.Subdir(prefix))
	if err != nil {
		// This shouldn't be possible, because we know the inner matcher is invertible.
		panic(err)
	}
	return i
}

func (m invertMatcher) Child(prefix string) Matcher {
	i, err := InvertMatcher(m.matcher.Child(prefix))
	if err != nil {
		// This shouldn't be possible, because we know the inner matcher is invertible.
		panic(err)
	}
	return i
}

// ANDs/ORs a bunch of matchers together.
type listMatcher struct {
	conjunction bool // If true, this is an AND. Otherwise it's an OR.
	matchers    []Matcher
}

func newListMatcher(conjunction bool, matchers []Matcher) Matcher {
	simplified := make([]Matcher, 0, len(matchers))
	for _, m := range matchers {
		if conjunction {
			if m.Empty() {
				return m
			} else if m.All() {
				continue
			}
		} else {
			if m.Empty() {
				continue
			} else if m.All() {
				return m
			}
		}
		simplified = append(simplified, m)
	}
	if len(simplified) == 1 {
		return simplified[0]
	}
	return listMatcher{conjunction: conjunction, matchers: simplified}
}

func newDisjunctionMatcher(matchers []Matcher) Matcher {
	return newListMatcher(false, matchers)
}

func newConjunctionMatcher(matchers []Matcher) Matcher {
	return newListMatcher(true, matchers)
}

func (d listMatcher) ToPatterns() []string {
	if d.All() {
		return []string{"**"}
	}

	result := make([]string, 0, len(d.matchers))
	for _, matcher := range d.matchers {
		result = append(result, matcher.ToPatterns()...)
	}
	return result
}

func (d listMatcher) Match(s string) bool {
	if d.conjunction {
		for _, matcher := range d.matchers {
			ok := matcher.Match(s)
			if !ok {
				return false
			}
		}
		return true
	} else {
		for _, matcher := range d.matchers {
			ok := matcher.Match(s)
			if ok {
				return true
			}
		}
		return false
	}
}

func (d listMatcher) Empty() bool {
	if d.conjunction {
		for _, matcher := range d.matchers {
			ok := matcher.Empty()
			if ok {
				return true
			}
		}
		return false
	} else {
		for _, matcher := range d.matchers {
			ok := matcher.Empty()
			if !ok {
				return false
			}
		}
		return true
	}
}

func (d listMatcher) All() bool {
	if d.conjunction {
		for _, matcher := range d.matchers {
			ok := matcher.All()
			if !ok {
				return false
			}
		}
		return true
	} else {
		for _, matcher := range d.matchers {
			ok := matcher.All()
			if ok {
				return true
			}
		}
		return false
	}
}

func (d listMatcher) IsNormal() bool {
	for _, m := range d.matchers {
		if !m.IsNormal() {
			return false
		}

		// Conjunctions may have inner lists, but they must be disjunctions.
		// Disjunctions may not have inner lists.
		innerList, isInnerList := m.(listMatcher)
		if isInnerList && !(d.conjunction && !innerList.conjunction) {
			return false
		}

		// Disjunctions may not have inner inversions
		if !d.conjunction {
			_, isInversion := m.(invertMatcher)
			if isInversion {
				return false
			}
		}
	}
	return true
}

func (d listMatcher) AsFileSet() []string {
	if d.conjunction {
		return nil
	}
	result := []string{}
	for _, m := range d.matchers {
		fileSet := m.AsFileSet()
		if fileSet == nil {
			return nil
		}
		result = append(result, fileSet...)
	}
	return result
}

func (d listMatcher) Subdir(prefix string) Matcher {
	matchers := make([]Matcher, len(d.matchers))
	for i, m := range d.matchers {
		matchers[i] = m.Subdir(prefix)
	}
	return newListMatcher(d.conjunction, matchers)
}

func (d listMatcher) Child(prefix string) Matcher {
	matchers := make([]Matcher, len(d.matchers))
	for i, m := range d.matchers {
		matchers[i] = m.Child(prefix)
	}
	return newListMatcher(d.conjunction, matchers)
}

// Matches a single file.
type fileMatcher struct {
	util PathUtil
	file string
}

const filePrefix = "file://"

func (m fileMatcher) ToPatterns() []string {
	return []string{filePrefix + m.file}
}

func (m fileMatcher) Match(path string) bool {
	return m.file == path
}

func (m fileMatcher) Empty() bool {
	return false
}

func (m fileMatcher) All() bool {
	return false
}

func (m fileMatcher) IsNormal() bool {
	return true
}

func (m fileMatcher) AsFileSet() []string {
	return []string{m.file}
}

func (m fileMatcher) Subdir(prefix string) Matcher {
	return fileMatcher{
		util: m.util,
		file: m.util.Join(prefix, m.file),
	}
}

func (m fileMatcher) Child(prefix string) Matcher {
	child, ok := Child(m.util, prefix, m.file)
	if !ok {
		return NewEmptyMatcher()
	}
	return fileMatcher{
		util: m.util,
		file: child,
	}
}

// Matches file paths.
//
// Pattern semantics attempt to match  `ls`. All patterns are taken
// relative to the root of the current directory.
//
// Uses ** globs for recursive matches.
//
// Implemented with golang's path.Match on each part of the path.
//
// Examples:
// 'foo' will match 'foo', but not 'foo/bar'
// 'foo/bar' will match 'foo/bar/baz' but not 'baz/foo/bar'
// '*.txt' will match 'foo.txt' and 'bar.baz.txt' but not 'foo/bar.txt'
// '*/foo.txt' will match 'a/foo.txt' but not 'foo.txt' or 'a/b/foo.txt'
// **/*.txt will match foo.txt and a/b/c/foo.txt
type patternMatcher struct {
	util    PathUtil
	pattern string
}

func (m patternMatcher) ToPatterns() []string {
	return []string{m.pattern}
}

func (m patternMatcher) Match(path string) bool {
	return m.matchRecur(m.pattern, path)
}

func (m patternMatcher) matchRecur(pattern string, path string) bool {
	// Base case #1: the pattern and path are both exhausted.
	if (pattern == "" || pattern == "**") && path == "" {
		return true
	}

	if pattern == "" {
		return false
	}

	// Base case #2: the path has been exhausted but there's still pattern
	// left to match.
	if path == "" {
		return false
	}

	pFirst, pRest := SplitFirst(m.util, pattern)
	first, rest := SplitFirst(m.util, path)
	if pFirst == "**" {
		// The double star case is special.
		// First recur on the case where the double star matches nothing.
		match := m.matchRecur(pRest, first)
		if match {
			return true
		}

		// If that doesn't match, recur on the case where the double star
		// matches the first part of the path.
		// Note that this is potentially exponential, and a "optimized" algorithm
		// would use a dynamic programming approach, but this is ok
		// for most cases.
		return m.matchRecur(pattern, rest)
	}

	// Normal patterns only match one part of the path.
	match, err := m.util.Match(pFirst, first)
	if err != nil {
		// The pattern should have been validated up-front.
		panic(err)
	}

	if !match {
		return false
	}

	// Recur on the next part of both the pattern and the path.
	return m.matchRecur(pRest, rest)
}

func (m patternMatcher) Empty() bool {
	return false
}

func (m patternMatcher) All() bool {
	return false
}

func (m patternMatcher) IsNormal() bool {
	return true
}

func (m patternMatcher) AsFileSet() []string {
	return nil
}

func (m patternMatcher) Subdir(prefix string) Matcher {
	return &patternMatcher{
		util:    m.util,
		pattern: m.util.Join(prefix, m.pattern),
	}
}

func (m patternMatcher) Child(prefix string) Matcher {
	child, ok := childPattern(m.util, prefix, m.pattern)
	if !ok {
		return NewEmptyMatcher()
	}
	result, err := NewMatcherFromPattern(m.util, child)
	if err != nil {
		panic(fmt.Sprintf("Child(%v, %s) produced invalid pattern: %q", m.ToPatterns(), prefix, child))
	}
	return result
}

// Matches nothing.
func NewEmptyMatcher() Matcher {
	return listMatcher{conjunction: false, matchers: []Matcher{}}
}

// Matches everything.
func NewAllMatcher() Matcher {
	return listMatcher{conjunction: true, matchers: []Matcher{}}
}

// Matches a single file only
func NewFileMatcher(util PathUtil, file string) (Matcher, error) {
	if file == "" {
		return nil, fmt.Errorf("NewFileMatcher: no file specified")
	}
	return fileMatcher{util: util, file: file}, nil
}

func NewFilesMatcher(util PathUtil, files []string) (Matcher, error) {
	matchers := make([]Matcher, 0, len(files))
	for _, f := range files {
		m, err := NewFileMatcher(util, f)
		if err != nil {
			return nil, err
		}
		matchers = append(matchers, m)
	}
	return newDisjunctionMatcher(matchers), nil
}

func NewMatcherFromPattern(util PathUtil, pattern string) (Matcher, error) {
	if strings.IndexFunc(pattern, unicode.IsSpace) != -1 {
		return nil, fmt.Errorf("Path patterns may not contain whitespace: %q", pattern)
	}

	if strings.HasPrefix(pattern, "/") {
		return nil, fmt.Errorf("Path patterns may not start with a leading slash: %q", pattern)
	}

	if isInverted(pattern) {
		inner, err := NewMatcherFromPattern(util, pattern[1:])
		if err != nil {
			return nil, err
		}
		return InvertMatcher(inner)
	}

	if strings.Index(pattern, filePrefix) == 0 {
		return NewFileMatcher(util, pattern[len(filePrefix):])
	}

	if pattern == "**" {
		return NewAllMatcher(), nil
	}

	// Validate the match pattern.
	// The only possible error from filepatch.Match is ErrBadPattern.
	_, err := util.Match(pattern, "")
	if err != nil {
		return nil, fmt.Errorf("Bad match pattern %q: %v", pattern, err)
	}

	return &patternMatcher{
		util:    util,
		pattern: pattern,
	}, nil
}

// When we have positive and negative patterns in the same pattern set,
// we treat them as a conjunction of all the positive forms, then disjunction on
// all the negative forms.
//
// For example, the pattern set [A, B, !C, !D] is interpreted as
// (A or B) and (not C) and (not D)
// We consider this Normal Form.
//
// We try to enforce that all matchers are in normal form, and reject matchers that are not.
func NewMatcherFromPatterns(util PathUtil, patterns []string) (Matcher, error) {
	positivePatterns := make([]string, 0, len(patterns))
	negativePatterns := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		if isInverted(pattern) {
			negativePatterns = append(negativePatterns, pattern)
		} else {
			positivePatterns = append(positivePatterns, pattern)
		}
	}

	positivePatterns, negativePatterns = simplifyPatterns(util, positivePatterns, negativePatterns)

	matchers := make([]Matcher, len(positivePatterns))
	for i, pattern := range positivePatterns {
		m, err := NewMatcherFromPattern(util, pattern)
		if err != nil {
			return nil, err
		}
		matchers[i] = m
	}

	invMatchers := make([]Matcher, len(negativePatterns))
	for i, pattern := range negativePatterns {
		m, err := NewMatcherFromPattern(util, pattern)
		if err != nil {
			return nil, err
		}
		invMatchers[i] = m
	}

	if len(matchers) != 0 {
		return newConjunctionMatcher(
				append([]Matcher{newDisjunctionMatcher(matchers)}, invMatchers...)),
			nil
	} else {
		return newConjunctionMatcher(invMatchers), nil
	}
}

func isInverted(p string) bool {
	return len(p) != 0 && p[0] == '!'
}

func MatchersEqual(a, b Matcher) bool {
	aPatterns := a.ToPatterns()
	bPatterns := b.ToPatterns()
	if len(aPatterns) != len(bPatterns) {
		return false
	}

	sort.Strings(aPatterns)
	sort.Strings(bPatterns)
	for i, aPattern := range aPatterns {
		bPattern := bPatterns[i]
		if aPattern != bPattern {
			return false
		}
	}
	return true
}

// Helper function to check if two positive patterns are orthogonal.
// By "orthogonal", we mean that there does not exist a path that can satisfy both.
func arePatternsOrthogonal(util PathUtil, p1, p2 string) bool {
	// This is a very simple algorithm that goes through each
	// path segment and see if they don't match.
	//
	// For example,
	// a/b/*
	// a/c/*
	// are not equal when we compare "b" and "c", so they are orthogonal.
	//
	// If we see any stars, or if we're out of path segments, we end immediately.
	p1First, p1Rest := SplitFirst(util, p1)
	if p1Rest == "" || strings.ContainsRune(p1First, '*') {
		return false
	}

	p2First, p2Rest := SplitFirst(util, p2)
	if p2Rest == "" || strings.ContainsRune(p2First, '*') {
		return false
	}

	if p1First != p2First {
		return true
	}
	return arePatternsOrthogonal(util, p1Rest, p2Rest)
}

// Helper to filter out negative patterns that are orthogonal
// to the positive patterns. As an example, if we have:
// ["*.txt", "!*.py"]
// we can skip the *.py.
//
// This is both an optimization and needed for correctness,
// because ["*.txt"] is invertible in our matcher engine
// but ["*.txt", "!*.py"] is not.
func simplifyPatterns(util PathUtil, positivePatterns, negativePatterns []string) ([]string, []string) {
	if len(positivePatterns) > 0 && len(negativePatterns) > 0 {
		simplifiedNegativePatterns := make([]string, 0, len(negativePatterns))
		for _, negPattern := range negativePatterns {
			p := negPattern[1:] // remove the "!"
			for _, posPattern := range positivePatterns {
				if !arePatternsOrthogonal(util, p, posPattern) {
					simplifiedNegativePatterns = append(simplifiedNegativePatterns, negPattern)
					break
				}
			}
		}
		return positivePatterns, simplifiedNegativePatterns
	}
	return positivePatterns, negativePatterns
}
