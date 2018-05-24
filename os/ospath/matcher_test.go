package ospath

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/windmilleng/mish/data/pathutil"
)

func TestGitPattern(t *testing.T) {
	m, _ := NewMatcherFromPattern(".git/**")
	assertMatch(t, m, ".git")
	assertMatch(t, m, ".git/a")
	assertMatch(t, m, ".git/a/b")
	assertNotMatch(t, m, "a/.git")
	assertNotMatch(t, m, "a/.git/b")
	assertNotMatch(t, m, "a/b/.git")
	assertNotMatch(t, m, ".gitx")
	assertNotMatch(t, m, "x.git")
	assertNotMatch(t, m, "a/.gitx")
}

func TestGitFile(t *testing.T) {
	m, _ := NewFileMatcher(".git")
	assertMatch(t, m, ".git")
	assertNotMatch(t, m, "a/.git")
}

func TestGitFilePattern(t *testing.T) {
	m, _ := NewMatcherFromPattern(".git")
	assertMatch(t, m, ".git")
	assertNotMatch(t, m, "a/.git")
}

func TestFolderStar(t *testing.T) {
	m, _ := NewMatcherFromPattern("*/x")
	assertMatch(t, m, "a/x")
	assertMatch(t, m, "b/x")
	assertNotMatch(t, m, "x")
	assertNotMatch(t, m, "a/b/x")
}

func TestEndStar(t *testing.T) {
	m, _ := NewMatcherFromPattern("a/*")
	assertMatch(t, m, "a/x")
	assertMatch(t, m, "a/y")
	assertNotMatch(t, m, "a")
	assertNotMatch(t, m, "x/a")
	assertNotMatch(t, m, "x/a/x")
	assertNotMatch(t, m, "a/x/y")
}

func TestDoubleStar(t *testing.T) {
	m, _ := NewMatcherFromPattern("**/*.go")
	assertMatch(t, m, "x.go")
	assertMatch(t, m, "a/x.go")
	assertMatch(t, m, "a/b/x.go")
	assertMatch(t, m, "a.go/x")
	assertNotMatch(t, m, "x")
	assertNotMatch(t, m, "x.gox")
}

func TestInnerDoubleStar(t *testing.T) {
	m, _ := NewMatcherFromPattern("a/**/*.go")
	assertMatch(t, m, "a/x.go")
	assertMatch(t, m, "a/b/x.go")
	assertMatch(t, m, "a/b/c/x.go")
	assertNotMatch(t, m, "x.go")
}

func TestSeparatorPattern(t *testing.T) {
	m, _ := NewMatcherFromPattern("a/b")
	assertMatch(t, m, "a/b")
	assertNotMatch(t, m, "a/b/c")
	assertNotMatch(t, m, "c/a/b")
	assertNotMatch(t, m, "a/bob")
}

func TestStarPattern(t *testing.T) {
	m, _ := NewMatcherFromPattern("*.txt")
	assertMatch(t, m, "a.txt")
	assertMatch(t, m, "b.txt")
	assertNotMatch(t, m, "a/txt")
}

func TestMultiplePattern(t *testing.T) {
	m, _ := NewMatcherFromPatterns([]string{"a", "b"})
	assertMatch(t, m, "a")
	assertMatch(t, m, "b")
	assertNotMatch(t, m, "a/b")
	assertNotMatch(t, m, "ab")
}

func TestSubdirPattern(t *testing.T) {
	m, _ := NewMatcherFromPattern("b/**")
	m = m.Subdir("a").(*Matcher)
	assertMatch(t, m, "a/b")
	assertMatch(t, m, "a/b/c")
	assertNotMatch(t, m, "c/a/b")
	assertNotMatch(t, m, "a/bob")
}

func TestChildPattern(t *testing.T) {
	m, _ := NewMatcherFromPattern("a/b/**")
	m = m.Child("a").(*Matcher)
	assertMatch(t, m, "b")
	assertMatch(t, m, "b/c")
	assertNotMatch(t, m, "a/b")
	assertNotMatch(t, m, "a/c")
}

func TestChildPatternEmpty(t *testing.T) {
	m, _ := NewMatcherFromPattern("a/b/c")
	m = m.Child("a/c").(*Matcher)
	if !m.Empty() {
		t.Errorf("Matcher %v should be empty", m)
	}
}

func TestChildPatternStar(t *testing.T) {
	m, _ := NewMatcherFromPattern("a/*/b")
	m = m.Child("a/x").(*Matcher)
	assertMatch(t, m, "b")
	assertNotMatch(t, m, "b/c")
	assertNotMatch(t, m, "a/b")
	assertNotMatch(t, m, "a/c")
}

func TestChildPatternStarEmpty(t *testing.T) {
	m, _ := NewMatcherFromPattern("a/*/b")
	m = m.Child("c").(*Matcher)
	assertNotMatch(t, m, "b")
	if !m.Empty() {
		t.Errorf("Matcher %v should be empty", m)
	}
}

func TestInversion(t *testing.T) {
	m, _ := NewMatcherFromPattern("!a/**")
	assertMatch(t, m, "b")
	assertMatch(t, m, "b/c")
	assertMatch(t, m, "b/a")
	assertMatch(t, m, "ax")
	assertMatch(t, m, "ax/ax")
	assertNotMatch(t, m, "a")
	assertNotMatch(t, m, "a/b")
}

func TestPositivesAndNegativesInversion(t *testing.T) {
	m, _ := NewMatcherFromPatterns([]string{"**/*.go", "!vendor/**"})
	assertMatch(t, m, "a.go")
	assertMatch(t, m, "b/a.go")
	assertNotMatch(t, m, "vendor/a.go")
}

func TestPositivesAndNegativesInversion2(t *testing.T) {
	m, _ := NewMatcherFromPatterns([]string{"**/*.go", "!vendor/**", "**/*.js", "!node_modules/**"})
	assertMatch(t, m, "a.go")
	assertMatch(t, m, "b/a.go")
	assertMatch(t, m, "a.js")
	assertMatch(t, m, "b/a.js")
	assertNotMatch(t, m, "vendor/a.go")
	assertNotMatch(t, m, "node_modules/b.js")
}

func TestInvertMatcher1(t *testing.T) {
	m, _ := NewMatcherFromPatterns([]string{"**/*.go", "!vendor/**"})
	m2, err := InvertMatcher(m)
	expected := "Inverted matcher cannot be written in normal form: [**/*.go !vendor/**]"
	if err == nil || err.Error() != expected {
		t.Errorf("Unexpected error: %v, %s", err, m2.ToPatterns())
	}
}

func TestInvertMatcher2(t *testing.T) {
	m, _ := NewMatcherFromPatterns([]string{"!**/*.go", "!vendor/**"})
	im, err := InvertMatcher(m)
	if err != nil {
		t.Fatal(err)
	}
	assertNotMatch(t, m, "a.go")
	assertMatch(t, im, "a.go")
	assertNotMatch(t, m, "vendor/a.js")
	assertMatch(t, im, "vendor/a.js")
	assertMatch(t, m, "a.js")
	assertNotMatch(t, im, "a.js")
}

func TestInvertMatcher3(t *testing.T) {
	m, _ := NewMatcherFromPatterns([]string{"**/*.go", "vendor/**"})
	im, err := InvertMatcher(m)
	if err != nil {
		t.Fatal(err)
	}
	assertMatch(t, m, "a.go")
	assertNotMatch(t, im, "a.go")
	assertMatch(t, m, "vendor/a.js")
	assertNotMatch(t, im, "vendor/a.js")
	assertNotMatch(t, m, "a.js")
	assertMatch(t, im, "a.js")
}

func TestSelfPattern(t *testing.T) {
	// The empty string is a special pattern that matches the current directory
	// but nothing below it, much like "." in Linux.
	m, _ := NewMatcherFromPattern("")
	assertNotEmpty(t, m)
	assertNotAll(t, m)
	assertMatch(t, m, "")
	assertNotMatch(t, m, "log.txt")
}

func TestAllAndNotAll(t *testing.T) {
	m, _ := NewMatcherFromPatterns([]string{"**", "!**"})
	assertEmpty(t, m)
	assertNotAll(t, m)
	assertNotMatch(t, m, "")
	assertNotMatch(t, m, "log.txt")
}

func TestAllAndNotSelf(t *testing.T) {
	m, _ := NewMatcherFromPatterns([]string{"**", "!"})
	assertNotEmpty(t, m)
	assertNotAll(t, m)
	assertNotMatch(t, m, "")
	assertMatch(t, m, "log.txt")
}

func TestAllAndChildEmpty(t *testing.T) {
	m, _ := NewMatcherFromPatterns([]string{"**", "!foo.txt"})
	assertNotEmpty(t, m)
	assertNotAll(t, m)
	assertMatch(t, m, "log.txt")
	assertNotMatch(t, m, "foo.txt")

	m = m.ChildOS("foo.txt")
	assertNotEmpty(t, m)
	assertNotAll(t, m)
	assertNotMatch(t, m, "")
	assertMatch(t, m, "log.txt")
	assertMatch(t, m, "foo.txt")
}

func TestChildDoubleStar(t *testing.T) {
	m, _ := NewMatcherFromPatterns([]string{"**/*.pb.go", "!vendor/**"})
	assertNotEmpty(t, m)
	assertMatch(t, m, "a.pb.go")
	assertMatch(t, m, "foo/bar/a.pb.go")
	assertNotMatch(t, m, "vendor/foo/a.pb.go")
	assertNotMatch(t, m, "a.go")

	child := m.ChildOS("foo")
	assertMatch(t, child, "a.pb.go")
	assertMatch(t, child, "foo/bar/a.pb.go")
	assertMatch(t, child, "vendor/foo/a.pb.go")
	assertNotMatch(t, child, "a.go")
}

func TestChildDoubleStarEmpty(t *testing.T) {
	m, _ := NewMatcherFromPatterns([]string{"!build/**"})
	assertNotEmpty(t, m)
	assertNotMatch(t, m, "build/log.txt")
	assertMatch(t, m, "log.txt")

	child := m.ChildOS("build")
	assertNotMatch(t, child, "log.go")
	assertEmpty(t, child)
}

func TestDoubleChildDoubleStar(t *testing.T) {
	m, _ := NewMatcherFromPatterns([]string{"pkg/**"})
	assertNotEmpty(t, m)
	assertMatch(t, m, "pkg/a")
	assertMatch(t, m, "pkg/a/b")
	assertMatch(t, m, "pkg/a/b/c")
	assertNotMatch(t, m, "log.txt")
	assertNotAll(t, m)
	assertNotEmpty(t, m)

	child := m.ChildOS("pkg")
	assertMatch(t, child, "a")
	assertMatch(t, child, "a/b")
	assertMatch(t, child, "a/b/c")
	assertAll(t, child)
	assertNotEmpty(t, child)

	child = m.ChildOS("pkg/a")
	assertMatch(t, child, "a")
	assertMatch(t, child, "a/b")
	assertMatch(t, child, "a/b/c")
	assertAll(t, child)
	assertNotEmpty(t, child)

	child = m.ChildOS("pkg/a/b/c")
	assertMatch(t, child, "a")
	assertMatch(t, child, "a/b")
	assertMatch(t, child, "a/b/c")
	assertAll(t, child)
	assertNotEmpty(t, child)
}

func TestJSON(t *testing.T) {
	matcher, _ := NewMatcherFromPattern("*/foo.txt")
	matcherBytes, err := json.Marshal(matcher)
	if err != nil {
		t.Fatal(err)
	}

	var newMatcher *Matcher
	err = json.Unmarshal(matcherBytes, &newMatcher)
	if err != nil {
		t.Fatal(err)
	}

	patterns := newMatcher.ToPatterns()
	if len(patterns) != 1 || patterns[0] != "*/foo.txt" {
		t.Errorf("Unexpected patterns after round trip: %v", patterns)
	}
}

func TestInvalidPattern(t *testing.T) {
	_, err := NewMatcherFromPattern("x x")
	if err == nil || !strings.Contains(err.Error(), "may not contain whitespace") {
		t.Errorf("Expected whitespace error. Actual: %v", err)
	}

	_, err = NewMatcherFromPattern("x\nx")
	if err == nil || !strings.Contains(err.Error(), "may not contain whitespace") {
		t.Errorf("Expected whitespace error. Actual: %v", err)
	}

	_, err = NewMatcherFromPattern("/x")
	if err == nil || !strings.Contains(err.Error(), "may not start with a leading slash") {
		t.Errorf("Expected leading slash error. Actual: %v", err)
	}
}

func TestSimplifyOrthogonalPatterns(t *testing.T) {
	m, _ := NewMatcherFromPatterns([]string{"a/b/*.go", "!a/c/*.go"})
	assertMatch(t, m, "a/b/c.go")
	assertNotMatch(t, m, "a/c/d.go")
	assertNotMatch(t, m, "a/d/e.go")

	inv, err := InvertMatcher(m)
	if err != nil {
		t.Fatal(err)
	}
	assertNotMatch(t, inv, "a/b/c.go")
	assertMatch(t, inv, "a/c/d.go")
	assertMatch(t, inv, "a/d/e.go")
}

type equal struct {
	a     *Matcher
	b     *Matcher
	equal bool
}

func expectEqual(t *testing.T, entry equal) {
	if entry.equal != MatchersEqual(entry.a, entry.b) {
		if entry.equal {
			t.Errorf("Expected matchers to be equal: (%q, %q)", entry.a, entry.b)
		} else {
			t.Errorf("Expected matchers to not be equal: (%q, %q)", entry.a, entry.b)
		}
	}
}

func TestEquals(t *testing.T) {
	m1, _ := NewMatcherFromPatterns([]string{"a"})
	m2, _ := NewMatcherFromPatterns([]string{"a", "b"})
	m3, _ := NewMatcherFromPatterns([]string{"b"})
	m4, _ := NewMatcherFromPatterns([]string{"b", "a"})
	expectEqual(t, equal{m1, m1, true})
	expectEqual(t, equal{m2, m2, true})
	expectEqual(t, equal{m3, m3, true})
	expectEqual(t, equal{m4, m4, true})
	expectEqual(t, equal{m1, m2, false})
	expectEqual(t, equal{m1, m3, false})
	expectEqual(t, equal{m1, m4, false})
	expectEqual(t, equal{m2, m3, false})
	expectEqual(t, equal{m2, m4, true})
}

func assertMatch(t *testing.T, m *Matcher, target string) {
	if !m.Match(target) {
		t.Errorf("Matcher %v should have matched %q", m.ToPatterns(), target)
	}
}

func assertNotMatch(t *testing.T, m *Matcher, target string) {
	if m.Match(target) {
		t.Errorf("Matcher %v should not have matched %q", m.ToPatterns(), target)
	}
}

func assertEmpty(t *testing.T, m pathutil.Matcher) {
	if !m.Empty() {
		t.Errorf("Matcher %v should have been empty", m.ToPatterns())
	}
}

func assertNotEmpty(t *testing.T, m pathutil.Matcher) {
	if m.Empty() {
		t.Errorf("Matcher %v should not have been empty", m.ToPatterns())
	}
}

func assertAll(t *testing.T, m pathutil.Matcher) {
	if !m.All() {
		t.Errorf("Matcher %v should have been an all matcher", m.ToPatterns())
	}
}

func assertNotAll(t *testing.T, m pathutil.Matcher) {
	if m.All() {
		t.Errorf("Matcher %v should not have been an all matcher", m.ToPatterns())
	}
}
