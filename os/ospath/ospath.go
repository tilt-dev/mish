package ospath

import (
	"os"
	"path/filepath"

	"github.com/windmilleng/mish/data/pathutil"
)

func Child(dir string, file string) (string, bool) {
	return pathutil.Child(theOsPathUtil, dir, file)
}

// Returns the absolute version of this path, resolving all symlinks.
func RealAbs(path string) (string, error) {
	// Make the path absolute first, so that we find any symlink parents.
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	// Resolve the symlinks.
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", err
	}

	// Double-check we're still absolute.
	return filepath.Abs(realPath)
}

func IsDir(path string) bool {
	f, err := os.Stat(path)
	if err != nil {
		return false
	}

	return f.Mode().IsDir()
}

type osPathUtil struct{}

func (p osPathUtil) Base(path string) string                  { return filepath.Base(path) }
func (p osPathUtil) Dir(path string) string                   { return filepath.Dir(path) }
func (p osPathUtil) Join(a, b string) string                  { return filepath.Join(a, b) }
func (p osPathUtil) Match(pattern, path string) (bool, error) { return filepath.Match(pattern, path) }
func (p osPathUtil) Separator() rune                          { return filepath.Separator }

var theOsPathUtil = osPathUtil{}
