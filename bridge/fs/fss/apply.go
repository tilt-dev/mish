package fss

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/errors"
	"github.com/windmilleng/mish/os/ospath"
)

// functions to apply ops to the filesystem

func canApply(op data.Op) bool {
	switch op.(type) {
	case *data.WriteFileOp, *data.RemoveFileOp, *data.InsertBytesFileOp, *data.DeleteBytesFileOp, *data.EditFileOp, *data.ChmodFileOp, *data.ClearOp:
		return true
	default:
		return false
	}
}

func applyToFS(op data.Op, checkout string) error {
	switch op := op.(type) {
	case *data.WriteFileOp:
		return applyWrite(op, checkout)
	case *data.RemoveFileOp:
		return applyRemove(op, checkout)
	case *data.InsertBytesFileOp:
		return applyInsert(op, checkout)
	case *data.DeleteBytesFileOp:
		return applyDelete(op, checkout)
	case *data.EditFileOp:
		return applyEdit(op, checkout)
	case *data.ChmodFileOp:
		return applyChmod(op, checkout)
	case *data.ClearOp:
		return applyClear(op, checkout)
	default:
		return fmt.Errorf("fss.applyToFS: unknown op type %T %v", op, op)
	}
}

func fileMode(executable bool) os.FileMode {
	if executable {
		return os.FileMode(0700)
	} else {
		return os.FileMode(0600)
	}
}

func applyWrite(op *data.WriteFileOp, checkout string) error {
	filePath := path.Join(checkout, op.Path)
	dirPath := path.Dir(filePath)
	err := os.MkdirAll(dirPath, os.FileMode(0700))
	if err != nil {
		return err
	}

	absPath := path.Join(checkout, op.Path)
	switch op.Type {
	case data.FileRegular:
		// if absPath is a symlink, we need to remove it before writing to it
		if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return ioutil.WriteFile(absPath, op.Data.InternalByteSlice(), fileMode(op.Executable))
	case data.FileSymlink:
		symlink := string(op.Data.InternalByteSlice())
		if filepath.IsAbs(symlink) {
			return fmt.Errorf("applyWrite: cannot handle absolute symlink")
		}

		target := filepath.Join(filepath.Dir(absPath), symlink)
		_, ok := ospath.Child(checkout, target)
		if !ok {
			return fmt.Errorf("applyWrite: symlinks can't point outside of root directory. Symlink %q points to %q", op.Path, symlink)
		}

		// Remove the newName if it exists.
		_, err := os.Lstat(absPath)
		if err == nil {
			if err := os.Remove(absPath); err != nil {
				return err
			}
		}
		if !os.IsNotExist(err) {
			return err
		}
		return os.Symlink(symlink, absPath)
	default:
		return fmt.Errorf("applyWrite: unknown file type %d: %s", op.Type, op.Path)
	}
}

func applyRemove(op *data.RemoveFileOp, checkout string) error {
	filePath := path.Join(checkout, op.Path)
	err := os.Remove(filePath)
	if err != nil {
		return err
	}

	// Check to see if any directories need to be removed.
	dir := path.Dir(filePath)

	// Don't traverse above the current dir, and be paranoid about it.
	for dir != checkout && len(dir) > len(checkout) {
		dirFile, err := os.Open(dir)
		if err != nil {
			return errors.Propagatef(err, "Error opening %s", dir)
		}

		entries, err := dirFile.Readdir(1)
		if err != nil && err != io.EOF {
			return errors.Propagatef(err, "Error reading %s", dir)
		}

		if len(entries) > 0 {
			break
		}

		err = os.Remove(dir)
		if err != nil {
			return errors.Propagatef(err, "Error removing %s", dir)
		}
		dir = path.Dir(dir)
	}
	return nil
}

func applyInsert(op *data.InsertBytesFileOp, checkout string) error {
	filePath := path.Join(checkout, op.Path)
	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.Propagatef(err, "Error reading %s", op.Path)
	}

	f, err := os.OpenFile(filePath, os.O_WRONLY, 0755)
	if err != nil {
		return errors.Propagatef(err, "Error opening file %s", op.Path)
	}
	defer f.Close()

	_, err = f.WriteAt(op.Data.InternalByteSlice(), op.Index)
	if err != nil {
		return errors.Propagatef(err, "Error writing file %s", op.Path)
	}

	if op.Index < int64(len(contents)) {
		_, err = f.WriteAt(contents[op.Index:], op.Index+int64(op.Data.Len()))
		if err != nil {
			return errors.Propagatef(err, "Error writing file %s", op.Path)
		}
	}

	return nil
}

func applyDelete(op *data.DeleteBytesFileOp, checkout string) error {
	filePath := path.Join(checkout, op.Path)
	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.Propagatef(err, "Error reading %s", op.Path)
	}

	f, err := os.OpenFile(filePath, os.O_WRONLY, 0755)
	if err != nil {
		return errors.Propagatef(err, "Error opening file %s", op.Path)
	}
	defer f.Close()

	_, err = f.WriteAt(contents[op.Index+op.DeleteCount:], op.Index)
	if err != nil {
		return errors.Propagatef(err, "Error writing file %s", op.Path)
	}

	return f.Truncate(int64(len(contents)) - op.DeleteCount)
}

func applyEdit(op *data.EditFileOp, checkout string) error {
	filePath := path.Join(checkout, op.Path)
	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.Propagatef(err, "Error reading %s", op.Path)
	}

	f, err := os.OpenFile(filePath, os.O_WRONLY, 0755)
	if err != nil {
		return errors.Propagatef(err, "Error opening file %s", op.Path)
	}
	defer f.Close()

	b := data.NewBytesWithBacking(contents).ApplySplices(op.Splices)
	_, err = f.WriteAt(b.InternalByteSlice(), 0)
	if err != nil {
		return errors.Propagatef(err, "Error writing file %s", op.Path)
	}
	return f.Truncate(int64(b.Len()))
}

func applyChmod(op *data.ChmodFileOp, checkout string) error {
	filePath := path.Join(checkout, op.Path)
	return os.Chmod(filePath, fileMode(op.Executable))
}

func applyClear(op *data.ClearOp, checkout string) error {
	// We don't want to remove the checkout, because then we couldn't write a file,
	// so we want to remove all files in this directory. So list them, and then remove each
	fs, err := ioutil.ReadDir(checkout)
	if err != nil {
		return err
	}

	for _, f := range fs {
		if err := os.RemoveAll(path.Join(checkout, f.Name())); err != nil {
			return err
		}
	}

	return nil
}
