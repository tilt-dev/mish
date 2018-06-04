package watch

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/windmilleng/fsnotify"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbint"
	"github.com/windmilleng/mish/errors"
	"github.com/windmilleng/mish/os/ospath"
)

var symlinkOutsideWorkspaceErr = fmt.Errorf("Symlink reaches outside workspace")

const defaultMaxFilesWatched = 1048576

type scanner struct {
	root    string
	matcher *ospath.Matcher

	// If a file is before the min mtime, we mark it ignored but still emit a trackedFile for it.
	minMtime time.Time

	watcher  wmNotify
	maxFiles int
}

func newScanner(root string, matcher *ospath.Matcher, watcher wmNotify, minMtime time.Time, max int) *scanner {
	return &scanner{root: root, matcher: matcher, watcher: watcher, minMtime: minMtime, maxFiles: max}
}

func (s *scanner) Root() string {
	return s.root
}

func (s *scanner) Matcher() *ospath.Matcher {
	return s.matcher
}

// Returns
// string: The path relative to the watch root
// bool: We should track this if it's a file.
// bool: We should track this if it's a directory
func (s *scanner) relPath(path string) (string, bool, bool) {
	relPath, err := filepath.Rel(s.root, path)
	if err != nil {
		return "", false, false
	}

	if relPath == "." {
		return "", true, true
	}

	if filepath.IsAbs(relPath) || strings.HasPrefix(relPath, "..") {
		return "", false, false
	}

	// Try to match this path
	trackIfFile := s.matcher.Match(relPath)

	// Check if we can match anything under this path.
	child := s.matcher.Child(relPath)
	trackIfDir := !child.Empty()

	return relPath, trackIfFile, trackIfDir
}

type trackedFile struct {
	absPath string
	relPath string
	mode    os.FileMode
	ignore  bool
}

// Converts this into the common SnapshotFile format.
//
// If the file has been deleted since this struct was created,
// returns a os.IsNotExist error.
//
// If the file is a symlink to somewhere outside the workspace,
// returns a symlinkOutsideWorkspaceErr.
func (f *trackedFile) toSnapshotFile() (*dbint.SnapshotFile, error) {
	var contents []byte
	fileType := data.FileRegular
	if f.mode.IsRegular() {
		var err error
		contents, err = ioutil.ReadFile(f.absPath)
		if err != nil {
			return nil, err
		}
	} else if isSymlink(f.mode) {
		link, err := os.Readlink(f.absPath)
		if err != nil {
			return nil, err
		}

		if filepath.IsAbs(link) {
			// For now, make all links relative when we convert them
			// to a Windmill SnapshotFile. In the future, we might have some notion
			// of an "absolute" link in the workspace.
			link, err = filepath.EvalSymlinks(link)
			if err != nil {
				return nil, err
			}

			link, err = filepath.Rel(filepath.Dir(f.absPath), link)
			if err != nil {
				return nil, err
			}
		}

		// Check to see if the link reaches outside the workspace.
		relToWorkspace := filepath.Join(f.relPath, link)
		if filepath.IsAbs(relToWorkspace) || strings.HasPrefix(relToWorkspace, ".") {
			return nil, symlinkOutsideWorkspaceErr
		}

		contents = []byte(link)
		fileType = data.FileSymlink
	} else {
		return nil, fmt.Errorf("toSnapshotFile: unexpected mode: %v", f.mode)
	}

	return &dbint.SnapshotFile{
		Path:       f.relPath,
		Contents:   data.NewBytesWithBacking(contents),
		Executable: data.IsExecutableMode(f.mode),
		Type:       fileType,
	}, nil
}

// helper for eventToUpdate
func handleStatError(relPath string, trackedIfFile bool, err error) (map[string]*trackedFile, error) {
	if !os.IsNotExist(err) {
		return nil, err
	}

	if !trackedIfFile {
		return nil, nil
	}

	return map[string]*trackedFile{relPath: nil}, nil
}

func (s *scanner) eventToUpdate(event fsnotify.Event) (map[string]*trackedFile, error) {
	path := event.Name

	relPath, trackedIfFile, trackedIfDir := s.relPath(path)
	if !trackedIfFile && !trackedIfDir {
		return nil, nil
	}

	mode := os.FileMode(0)

	st, err := os.Lstat(path)
	if err != nil {
		return handleStatError(relPath, trackedIfFile, err)
	}

	mode = st.Mode()
	if isSymlink(mode) {
		// Check to make sure the symlink is valid.
		// Emacs creates deliberately broken symlinks as lock files.
		_, err := os.Stat(path)
		if err != nil {
			return handleStatError(relPath, trackedIfFile, err)
		}
	}

	if mode.IsDir() && event.Op&fsnotify.Create != 0 {
		if !trackedIfDir {
			return nil, nil
		}
		return s.scanNewDir(path)
	}

	if isScannableFileMode(mode) {
		if !trackedIfFile {
			return nil, nil
		}

		f := &trackedFile{absPath: path, relPath: relPath, mode: mode}
		return map[string]*trackedFile{relPath: f}, nil
	}

	return nil, nil
}

// Walks a directory recursively and sets up fs watches.
// Returns all the paths we added.
func (s *scanner) scanNewDir(path string) (map[string]*trackedFile, error) {
	files := make(map[string]*trackedFile)

	err := filepath.Walk(path, func(innerPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, trackedIfFile, trackedIfDir := s.relPath(innerPath)
		if !trackedIfFile && !trackedIfDir {
			return nil
		}

		mode := info.Mode()
		if isSymlink(mode) {
			// Check to make sure the symlink is valid.
			// Emacs creates deliberately broken symlinks as lock files.
			_, err := os.Stat(innerPath)
			if err != nil {
				if !os.IsNotExist(err) {
					return err
				}

				if trackedIfFile {
					files[relPath] = nil
				}
				return nil
			}
		}

		if mode.IsDir() && !trackedIfDir {
			return filepath.SkipDir
		} else if !mode.IsDir() && !trackedIfFile {
			return nil
		}
		if s.watcher != nil && mode.IsDir() {
			// Dirs capture file events directly in the directively, but
			// not recursively in inner directories, so we need to watch each
			// inner directory.
			err = s.watcher.Add(innerPath)
			if err != nil {
				return errors.Propagatef(err, "scanNewDir")
			}
		}
		if isScannableFileMode(mode) {
			f := &trackedFile{absPath: innerPath, relPath: relPath, mode: mode}

			// minMtime tracking is just an optimization, so we should only
			// ignore file that are strictly before the minMtime.
			if info.ModTime().Before(s.minMtime) {
				f.ignore = true
			}

			files[relPath] = f
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	lenFiles := len(files)
	if lenFiles > s.maxFiles {
		return nil, fmt.Errorf("trying to watch too many files. Max is %d but tried to watch %d.", s.maxFiles, lenFiles)
	}

	return files, nil
}

func isScannableFileMode(mode os.FileMode) bool {
	return mode.IsRegular() || isSymlink(mode)
}

func isSymlink(mode os.FileMode) bool {
	return (mode & os.ModeSymlink) == os.ModeSymlink
}
