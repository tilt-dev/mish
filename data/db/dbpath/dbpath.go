package dbpath

import (
	"path"
	"strings"

	"github.com/windmilleng/mish/data/pathutil"
)

// The number of "hops" from fileA to fileB
// (e.g., 0 = same file, 1 = same directory, etc.)
func Distance(fileA string, fileB string) int {
	filePartsA := SplitAll(fileA)
	filePartsB := SplitAll(fileB)

	for i := 0; i < len(filePartsA); i++ {
		if i >= len(filePartsB) {
			return (len(filePartsA) - i)
		}

		if filePartsA[i] != filePartsB[i] {
			return (len(filePartsA) - i) + (len(filePartsB) - i) - 1
		}
	}

	return len(filePartsB) - len(filePartsA)
}

func Child(dir string, file string) (string, bool) {
	return pathutil.Child(theDbPathUtil, dir, file)
}

func Dir(dir string) string {
	r := path.Dir(dir)

	r = noPeriod(r)

	return r
}

func noPeriod(p string) string {
	if p == "." {
		return ""
	}
	return p
}

const slash = '/'

func clean(p string) string {
	return noPeriod(path.Clean(p))
}

func Split(p string) (string, string) {
	p = clean(p)
	firstSlash := strings.IndexRune(p, slash)
	if firstSlash == -1 {
		return p, ""
	}

	return p[0:firstSlash], p[firstSlash+1:]
}

func SplitAll(p string) []string {
	p = clean(p)
	result := make([]string, 0)

	var part string
	for {
		if p == "" {
			return result
		}

		part, p = Split(p)
		result = append(result, part)
	}
}
