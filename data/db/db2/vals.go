package db2

import (
	"fmt"
	"path"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbpath"
)

type snapshotVal interface {
	snapshotVal()

	contentID() data.SnapshotID

	opEvalResult
}

type snapshotDir struct {
	files map[string]snapshotVal

	// If we know the content ID of this dir, we store it here.
	// Notice that if the content ID of a file changes, we don't invalidate
	// all IDs of parents, so we double-check when we load.
	uncheckedContentID data.SnapshotID
}

func (d *snapshotDir) snapshotVal()  {}
func (d *snapshotDir) opEvalResult() {}

func (d *snapshotDir) contentID() data.SnapshotID {
	contentID := d.uncheckedContentID
	if contentID.Nil() {
		return data.SnapshotID{}
	}

	for _, f := range d.files {
		if f.contentID().Nil() {
			return data.SnapshotID{}
		}
	}
	return contentID
}

func diffPaths(v1, v2 snapshotVal) []string {
	return diffPathsHelper("", v1, v2, []string{})
}

func diffPathsHelper(curPath string, f1, f2 snapshotVal, acc []string) []string {
	switch f1 := f1.(type) {
	case *snapshotDir:
		switch f2 := f2.(type) {
		case *snapshotDir:
			return diffDirPathsHelper(curPath, f1, f2, acc)
		case *snapshotFile:
			acc = append(acc, curPath)
			return allPathsHelper(curPath, f1, acc)
		default:
			panic(fmt.Sprintf("Impossible snapshotVal %T", f2))
		}

	case *snapshotFile:
		switch f2 := f2.(type) {
		case *snapshotDir:
			acc = append(acc, curPath)
			return allPathsHelper(curPath, f2, acc)
		case *snapshotFile:
			sameContentID := !f1.cID.Nil() && f1.cID == f2.cID
			if sameContentID {
				return acc
			}

			sameContent := f1.data.Equal(f2.data)
			if sameContent {
				return acc
			}

			return append(acc, curPath)
		default:
			panic(fmt.Sprintf("Impossible snapshotVal %T", f2))
		}

	default:
		panic(fmt.Sprintf("Impossible snapshotVal %T", f1))
	}
}

func diffDirPathsHelper(curPath string, d1, d2 *snapshotDir, acc []string) []string {
	for k, f1 := range d1.files {
		curPath := path.Join(curPath, k)
		f2, ok := d2.files[k]
		if !ok {
			acc = allPathsHelper(curPath, f1, acc)
			continue
		}

		acc = diffPathsHelper(curPath, f1, f2, acc)
	}

	for k, f2 := range d2.files {
		_, ok := d1.files[k]
		if !ok {
			acc = allPathsHelper(path.Join(curPath, k), f2, acc)
		}
	}

	return acc
}

func allPathsHelper(curPath string, val snapshotVal, acc []string) []string {
	switch val := val.(type) {
	case *snapshotDir:
		for k, v := range val.files {
			acc = allPathsHelper(path.Join(curPath, k), v, acc)
		}
	case *snapshotFile:
		acc = append(acc, curPath)
	default:
		panic(fmt.Sprintf("Impossible snapshotVal %T", val))
	}
	return acc
}

func newDir() *snapshotDir {
	return &snapshotDir{files: make(map[string]snapshotVal)}
}

type snapshotFile struct {
	data       data.Bytes
	executable bool
	cID        data.SnapshotID
	fileType   data.FileType
}

func (f *snapshotFile) snapshotVal()  {}
func (f *snapshotFile) opEvalResult() {}

func (f *snapshotFile) contentID() data.SnapshotID {
	return f.cID
}

type onNotFound int

const (
	onNotFoundDoNothing onNotFound = iota
	onNotFoundCreate
	onNotFoundError
)

func pathSet(val snapshotVal) map[string]bool {
	paths := allPathsHelper("", val, []string{})
	result := make(map[string]bool, len(paths))
	for _, p := range paths {
		result[p] = true
	}
	return result
}

func lookup(root snapshotVal, origPath string, onNotFound onNotFound) (snapshotVal, error) {
	currentPath := ""
	remainingPath := origPath
	currentVal := root

	for remainingPath != "" {
		switch innerDir := currentVal.(type) {
		case *snapshotFile:
			return nil, fmt.Errorf("Lookup(%q): parent %q is a file not a directory", origPath, currentPath)
		case *snapshotDir:
			first, rest := dbpath.Split(remainingPath)
			currentPath = path.Join(currentPath, first)
			next, ok := innerDir.files[first]
			if !ok {
				switch onNotFound {
				case onNotFoundCreate:
					next = newDir()
					innerDir.files[first] = next
				case onNotFoundError:
					if rest == "" {
						return nil, fmt.Errorf("Lookup(%q): does not exist", origPath)
					} else {
						return nil, fmt.Errorf("Lookup(%q): parent %q does not exist", origPath, currentPath)
					}
				case onNotFoundDoNothing:
					return nil, nil
				default:
					return nil, fmt.Errorf("Unrecognized onNotFound: %v", onNotFound)
				}
			}
			currentVal = next
			remainingPath = rest
		}
	}
	return currentVal, nil
}

func lookupDir(root snapshotVal, p string, onNotFound onNotFound) (*snapshotDir, error) {
	val, err := lookup(root, p, onNotFound)
	if err != nil {
		return nil, err
	}

	if val == nil && onNotFound == onNotFoundDoNothing {
		return nil, nil
	}

	dir, ok := val.(*snapshotDir)
	if !ok {
		return nil, fmt.Errorf("loopupDir(%q): expected directory, found %v", p, val)
	}

	return dir, nil
}

func lookupFile(root snapshotVal, p string, onNotFound onNotFound) (*snapshotFile, error) {
	val, err := lookup(root, p, onNotFound)
	if err != nil {
		return nil, err
	}

	if val == nil && onNotFound == onNotFoundDoNothing {
		return nil, nil
	}

	dir, ok := val.(*snapshotFile)
	if !ok {
		return nil, fmt.Errorf("loopupFile(%q): expected file, found %v", p, val)
	}

	return dir, nil
}
