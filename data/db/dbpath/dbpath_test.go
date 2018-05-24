package dbpath

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestChild(t *testing.T) {
	expected := []struct {
		dir     string
		file    string
		rel     string
		matches bool
		error   bool
	}{
		{"", "bar", "bar", true, false},
		{"foo", "bar", "", false, false},
		{"foo", "foo/bar", "bar", true, false},
		{"foo", "food", "", false, false},
		{"", "foo/bar", "foo/bar", true, false},
		{"foo/bar", "foo/bar/baz.txt", "baz.txt", true, false},
		{"foo.txt", "foo.txt", "", true, false},

		{"/foo", "bar", "", false, true},
	}

	for _, e := range expected {
		t.Run(fmt.Sprintf("%v", e), func(t *testing.T) {
			s, b := Child(e.dir, e.file)

			if s != e.rel || b != e.matches {
				toPrint := "nil"
				if e.error {
					toPrint = "<error>"
				}
				t.Fatalf("got: %q, %v expected %v, %v, %v", s, b, e.rel, e.matches, toPrint)
			}
		})
	}
}

func TestSplit(t *testing.T) {
	expected := []struct {
		in   string
		out1 string
		out2 string
	}{
		{"", "", ""},
		{"foo", "foo", ""},
		{"foo/", "foo", ""},
		{"foo/bar", "foo", "bar"},
		{"foo/bar/", "foo", "bar"},
	}

	for _, e := range expected {
		t.Run(fmt.Sprintf(e.in), func(t *testing.T) {
			o1, o2 := Split(e.in)
			if o1 != e.out1 || o2 != e.out2 {
				t.Fatalf("got: %q %q expected: %q %q", o1, o2, e.out1, e.out2)
			}
		})
	}
}

func TestDistance(t *testing.T) {
	expected := []struct {
		pathA    string
		pathB    string
		distance int
	}{
		{"foo.txt", "foo.txt", 0},
		{"a/foo.txt", "a/foo.txt", 0},
		{"a/b/foo.txt", "a/b/foo.txt", 0},
		{"a/b/foo.txt", "a/b/bar.txt", 1},
		{"a/b/foo.txt", "a/b", 1},
		{"a/b/foo.txt", "a", 2},
		{"a/b/foo.txt", "a/bar.txt", 2},
		{"a/b/foo.txt", "a/c/bar.txt", 3},
	}

	for _, e := range expected {
		t.Run(fmt.Sprintf("%v", e), func(t *testing.T) {
			actual := Distance(e.pathA, e.pathB)

			if actual != e.distance {
				t.Fatalf("distance(%s, %s) == %d. Expected: %d",
					e.pathA, e.pathB, actual, e.distance)
			}

			// Assert symmetry
			actual = Distance(e.pathB, e.pathA)

			if actual != e.distance {
				t.Fatalf("distance(%s, %s) not symmetric. One direction: %d. Other direction: %d",
					e.pathA, e.pathB, actual, e.distance)
			}
		})
	}
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
