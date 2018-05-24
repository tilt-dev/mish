package dbint

import (
	"path"
)

func PathSetToPaths(set map[string]bool) []string {
	paths := make([]string, 0, len(set))
	for k, _ := range set {
		paths = append(paths, k)
	}
	return paths
}

func PathsToPathSet(paths []string) map[string]bool {
	set := make(map[string]bool, len(paths))
	for _, path := range paths {
		set[path] = true
	}
	return set
}

type FileMap map[string]*SnapshotFile

func FilesToFileMapByBase(files []*SnapshotFile) FileMap {
	m := make(FileMap, len(files))
	for _, f := range files {
		m[path.Base(f.Path)] = f
	}
	return m
}
