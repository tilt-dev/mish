package db2

import (
	"fmt"
	"path"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbpath"
	"github.com/windmilleng/mish/errors"
)

// An evaluation engine that produces a snapshotVal.
type snapshotEvaluator struct {
	// If matcher is not empty, we will only bother calculating the value of this matcher,
	// and we'll short-circuit the iteration backwards if we can.
	matcher *dbpath.Matcher

	stripContents bool
}

func (d *snapshotEvaluator) empty() opEvalResult {
	return newDir()
}

func (d *snapshotEvaluator) visitBackwards(op data.Op, input data.SnapshotID, inputNum int) (opEvaluator, opEvalResult, error) {
	if d.matcher.Empty() {
		return d, nil, nil
	}

	matcher, err := visitBackwardsAdjustingPath(d.matcher, op, inputNum)
	if err != nil || matcher.Empty() {
		return nil, nil, err
	}

	// If this is a WriteFile or RemoveFile and we're only getting one file's
	// contents, there's no need to traverse backwards.
	// (This also needs special handling on post-traversal.)
	paths := matcher.AsFileSet()
	if len(paths) == 1 {
		path := paths[0]
		writeFileOp, ok := op.(*data.WriteFileOp)
		if ok && writeFileOp.FilePath() == path {
			return nil, nil, nil
		}

		removeFileOp, ok := op.(*data.RemoveFileOp)
		if ok && removeFileOp.FilePath() == path {
			return nil, nil, nil
		}
	}

	stripContents := d.stripContents
	if !stripContents {
		preserveOp, ok := op.(*data.PreserveOp)
		if ok && preserveOp.StripContents {
			stripContents = true
		}
	}

	return &snapshotEvaluator{matcher: matcher, stripContents: stripContents}, nil, nil
}

func (d *snapshotEvaluator) applyOp(outID data.SnapshotID, op data.Op, genericInputs []opEvalResult) (opEvalResult, error) {
	inputs := make([]snapshotVal, 0, len(genericInputs))
	for _, gi := range genericInputs {
		i, ok := gi.(snapshotVal)
		if !ok {
			// If we reach this, there's a bad bug in snapshotEvaluator.
			panic(fmt.Errorf("applyOp: unexpected opEvalResult type %T %v", gi, gi))
		}
		inputs = append(inputs, i)
	}

	if isLinearOp(op) {
		var input snapshotVal
		switch len(inputs) {
		case 0:
			input = newDir()
		case 1:
			input = inputs[0].(snapshotVal)
		default:
			return nil, fmt.Errorf("applyOp: non-linear inputs for op %v: %v", op, inputs)
		}

		return d.applyLinearOp(outID, op, input)
	}

	switch op := op.(type) {
	case *data.DirOp:
		return d.applyDirOp(outID, op, inputs)
	case *data.OverlayOp:
		return d.applyOverlayOp(inputs)
	default:
		return nil, fmt.Errorf("applyOp: unexpected op type %T %v", op, op)
	}
}

func (d *snapshotEvaluator) applyLinearOp(outID data.SnapshotID, op data.Op, input snapshotVal) (snapshotVal, error) {
	fileOp, ok := op.(data.FileOp)
	if ok && !d.matcher.Match(fileOp.FilePath()) {
		return input, nil
	}

	switch op := op.(type) {
	case *data.WriteFileOp:
		return d.applyWriteFileOp(outID, op, input)
	case *data.RemoveFileOp:
		return d.applyRemoveFileOp(op, input)
	case *data.SubdirOp:
		return d.applySubdirOp(op, input)
	case *data.RmdirOp:
		return d.applyRmdirOp(op, input)
	case *data.PreserveOp:
		return d.applyPreserveOp(op, input)
	case *data.IdentityOp:
		return input, nil
	case data.FileOp:
		return d.applyFileModOp(op, input)
	case *data.FailureOp:
		return nil, fmt.Errorf("applyLinearOp: %s", op.Message)
	default:
		return nil, fmt.Errorf("applyLinearOp: unexpected op type %T %v", op, op)
	}
}

func isLinearOp(op data.Op) bool {
	switch op := op.(type) {
	case *data.WriteFileOp, *data.SubdirOp, *data.RemoveFileOp, *data.InsertBytesFileOp, *data.DeleteBytesFileOp, *data.EditFileOp, *data.ChmodFileOp, *data.RmdirOp, *data.PreserveOp, *data.IdentityOp, *data.FailureOp:
		return true
	case *data.DirOp:
		return false
	case *data.OverlayOp:
		return false
	default:
		panic(fmt.Errorf("isLinearOp: unexpected op type %T %v", op, op))
	}
}

// only to be used by data/cmd/cmds
func FriendIsLinearOp(op data.Op) bool {
	return isLinearOp(op)
}

func (d *snapshotEvaluator) applyWriteFileOp(outID data.SnapshotID, op *data.WriteFileOp, input snapshotVal) (snapshotVal, error) {
	file := &snapshotFile{
		data:       op.Data,
		executable: op.Executable,
		fileType:   op.Type,
	}

	if op.Path == "" {
		if outID.IsContentID() {
			file.cID = outID
		}
		return file, nil
	}

	if d.stripContents {
		file.data = data.NewEmptyBytes()
		file.cID = data.SnapshotID{}
	}

	dir, err := lookupDir(input, dbpath.Dir(op.Path), onNotFoundCreate)
	if err != nil {
		return nil, errors.Propagatef(err, "applyWriteFileOp")
	}

	dir.files[path.Base(op.Path)] = file
	return input, nil
}

func (d *snapshotEvaluator) applyRemoveFileOp(op *data.RemoveFileOp, input snapshotVal) (snapshotVal, error) {
	if op.Path == "" {
		return newDir(), nil
	}

	// Check for the case where we short-circuited the pre-traversal. We know the file isn't here.
	paths := d.matcher.AsFileSet()
	if len(paths) == 1 && op.Path == paths[0] {
		return input, nil
	}

	if err := removeFile(input, op.Path); err != nil {
		return nil, err
	}
	return input, nil
}

func (d *snapshotEvaluator) applyFileModOp(op data.FileOp, input snapshotVal) (snapshotVal, error) {
	f, err := lookupFile(input, op.FilePath(), onNotFoundError)
	if err != nil {
		return nil, errors.Propagatef(err, "applyFileModOp")
	}
	f.cID = data.SnapshotID{}

	if d.stripContents {
		return input, nil
	}

	switch op := op.(type) {
	case *data.InsertBytesFileOp:
		f.data = f.data.InsertAt(int(op.Index), op.Data)
	case *data.DeleteBytesFileOp:
		f.data = f.data.RemoveAt(int(op.Index), int(op.DeleteCount))
	case *data.EditFileOp:
		f.data = f.data.ApplySplices(op.Splices)
	case *data.ChmodFileOp:
		f.executable = op.Executable
	default:
		return nil, fmt.Errorf("applyFileModOp: unexpected op type: %T %v", op, op)
	}

	return input, nil
}

func (d *snapshotEvaluator) applyRmdirOp(op *data.RmdirOp, input snapshotVal) (snapshotVal, error) {
	if op.Path == "" {
		return newDir(), nil
	}
	dir, err := lookupDir(input, dbpath.Dir(op.Path), onNotFoundError)
	if err != nil {
		return nil, errors.Propagatef(err, "applyRmdirOp")
	}

	delete(dir.files, path.Base(op.Path))
	dir.uncheckedContentID = data.SnapshotID{}
	return input, nil
}

func (d *snapshotEvaluator) applySubdirOp(op *data.SubdirOp, input snapshotVal) (snapshotVal, error) {
	return lookup(input, op.Path, onNotFoundError)
}

func (d *snapshotEvaluator) applyDirOp(outID data.SnapshotID, op *data.DirOp, inputs []snapshotVal) (snapshotVal, error) {
	dir := newDir()
	for i, name := range op.Names {
		innerDirs := dbpath.SplitAll(name)
		currentDir := dir
		for j, innerDir := range innerDirs {
			if j < len(innerDirs)-1 {
				nextDir := newDir()
				currentDir.files[innerDir] = nextDir
				currentDir = nextDir
			} else {
				currentDir.files[innerDir] = inputs[i]
			}
		}
	}

	if outID.IsContentID() {
		dir.uncheckedContentID = outID
	}

	return dir, nil
}

func (d *snapshotEvaluator) applyOverlayOp(inputs []snapshotVal) (snapshotVal, error) {
	if len(inputs) == 1 {
		return inputs[0], nil
	}

	lower, uppers := inputs[0], inputs[1:]
	switch lower := lower.(type) {
	case *snapshotFile:
		// A file is always wiped out by whatever's on top.
		return uppers[len(uppers)-1], nil

	case *snapshotDir:
		for i := len(uppers) - 1; i >= 0; i-- {
			_, isFile := uppers[i].(*snapshotFile)
			if isFile {
				// A file always wipes out a directory.
				return d.applyOverlayOp(uppers[i:])
			}
		}

		overlayMap := map[string][]snapshotVal{}
		for _, input := range inputs {
			inputDir, ok := input.(*snapshotDir)
			if !ok {
				return nil, fmt.Errorf("Unexpected snapshot val %T", input)
			}

			for k, v := range inputDir.files {
				overlayMap[k] = append(overlayMap[k], v)
			}
		}

		for k, innerInputs := range overlayMap {
			innerOutput, err := d.applyOverlayOp(innerInputs)
			if err != nil {
				return nil, err
			}
			lower.files[k] = innerOutput
			lower.uncheckedContentID = data.SnapshotID{}
		}
	}
	return lower, nil
}

func (d *snapshotEvaluator) applyPreserveOp(op *data.PreserveOp, input snapshotVal) (snapshotVal, error) {
	_, applied := preTraversePreserveOp(d.matcher, op)
	if applied {
		return input, nil
	}

	r := pathSet(input)
	for k, _ := range r {
		if !op.Matcher.Match(k) {
			if err := removeFile(input, k); err != nil {
				return nil, err
			}
		}
	}
	return input, nil
}

func removeFile(input snapshotVal, p string) error {
	_, err := lookupFile(input, p, onNotFoundError)
	if err != nil {
		return errors.Propagatef(err, "removeFile")
	}

	dir, err := lookupDir(input, dbpath.Dir(p), onNotFoundError)
	if err != nil {
		return errors.Propagatef(err, "removeFile")
	}

	delete(dir.files, path.Base(p))
	dir.uncheckedContentID = data.SnapshotID{}
	return nil
}
