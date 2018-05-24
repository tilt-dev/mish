// Common utilities for DB paths and OS paths

package pathutil

import (
	"strings"
)

type PathUtil interface {
	Base(path string) string
	Dir(path string) string
	Join(dir, base string) string
	Match(pattern, name string) (matched bool, err error)
	Separator() rune
}

// Split the path into (firstDir, rest). Note that this is
// different from normal Split(), which splits things into (dir, base).
// SplitFirst is better for matching algorithms.
//
// If the path cannot be split, returns (p, "").
func SplitFirst(util PathUtil, p string) (string, string) {
	firstSlash := strings.IndexRune(p, util.Separator())
	if firstSlash == -1 {
		return p, ""
	}

	return p[0:firstSlash], p[firstSlash+1:]
}

// Given absolute paths `dir` and `file`, returns
// the relative path of `file` relative to `dir`.
//
// Returns true if successful. If `file` is not under `dir`, returns false.
func Child(util PathUtil, dir, file string) (string, bool) {
	current := file
	child := ""
	for true {
		if dir == current {
			return child, true
		}

		if len(current) <= len(dir) || current == "." {
			return "", false
		}

		cDir := util.Dir(current)
		cBase := util.Base(current)
		child = util.Join(cBase, child)
		current = cDir
	}

	return "", false
}

// Like Child(), but file is a pattern instead of a path.
func childPattern(util PathUtil, dir, filePattern string) (string, bool) {
	current := filePattern
	child := ""
	for true {
		matched := (&patternMatcher{util: util, pattern: current}).Match(dir)
		if matched {
			// The recursive glob (**) can match multiple directories.
			// So if current terminates in a recursive glob, then the child pattern
			// must also begin with a recursive glob.
			currentBase := util.Base(current)
			if currentBase == "**" {
				return util.Join("**", child), true
			}

			return child, true
		}

		if current == "." || current == "" {
			return "", false
		}

		cDir := util.Dir(current)
		cBase := util.Base(current)
		child = util.Join(cBase, child)
		current = cDir
	}

	return "", false
}
