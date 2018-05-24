package data

import (
	"fmt"
	"os"
	"strings"

	"github.com/windmilleng/mish/data/db/dbpath"
)

func IsExecutableMode(mode os.FileMode) bool {
	return mode&0100 == 0100
}

type FileType uint8

const (
	FileRegular FileType = iota

	// The contents of a symlink are a relative dbpath.
	FileSymlink
)

type WriteFileOp struct {
	Path       string
	Data       Bytes
	Executable bool
	Type       FileType
}

func (op *WriteFileOp) FilePath() string { return op.Path }

func (op *WriteFileOp) Op() {}

func (op *WriteFileOp) OpEqual(b Op) bool {
	op2, ok := b.(*WriteFileOp)
	if !ok {
		return false
	}
	return op.Path == op2.Path && op.Data.Equal(op2.Data) && op.Executable == op2.Executable && op.Type == op2.Type
}

func (op *WriteFileOp) String() string {
	if op.Type == FileSymlink {
		return fmt.Sprintf("WriteFileOp(%s, %s, Symlink)", op.Path, op.Data.String())
	}

	// To make debugging easier, we print the string if it's short.
	if op.Data.Len() < 20 {
		return fmt.Sprintf("WriteFileOp(%s, %q, %v)", op.Path, op.Data.String(), op.Executable)
	} else {
		return fmt.Sprintf("WriteFileOp(%s, bytes[%d], %v)", op.Path, op.Data.Len(), op.Executable)
	}
}

type RemoveFileOp struct {
	Path string
}

func (op *RemoveFileOp) FilePath() string { return op.Path }

func (op *RemoveFileOp) Op() {}

func (op *RemoveFileOp) OpEqual(b Op) bool {
	op2, ok := b.(*RemoveFileOp)

	if !ok {
		return false
	}

	return op.FilePath() == op2.FilePath()
}

func (op *RemoveFileOp) String() string {
	return fmt.Sprintf("RemoveFileOp(%s)", op.Path)
}

type EditFileOp struct {
	Path    string
	Splices []EditFileSplice
}

func (op *EditFileOp) FilePath() string { return op.Path }

func (op *EditFileOp) Op() {}

func (op *EditFileOp) OpEqual(b Op) bool {
	op2, ok := b.(*EditFileOp)
	if !ok {
		return false
	}
	if op.Path != op2.Path || len(op.Splices) != len(op2.Splices) {
		return false
	}

	for i, s := range op.Splices {
		if !s.SpliceEqual(op2.Splices[i]) {
			return false
		}
	}
	return true
}

func (op *EditFileOp) String() string {
	return fmt.Sprintf("EditFileOp(%s)", op.Path)
}

type EditFileSplice interface {
	Splice()
	SpliceEqual(s2 EditFileSplice) bool
}

type InsertBytesSplice struct {
	Index int64
	Data  Bytes
}

func (s *InsertBytesSplice) Splice() {}

func (s *InsertBytesSplice) SpliceEqual(b EditFileSplice) bool {
	s2, ok := b.(*InsertBytesSplice)
	if !ok {
		return false
	}
	return s.Index == s2.Index && s.Data.Equal(s2.Data)
}

func (s *InsertBytesSplice) String() string {
	// To make debugging easier, we print the string if it's short.
	if s.Data.Len() < 20 {
		return fmt.Sprintf("Insert(%d, %q)", s.Index, s.Data.String())
	} else {
		return fmt.Sprintf("Insert(%d, bytes[%d])", s.Index, s.Data.Len())
	}
}

type DeleteBytesSplice struct {
	Index       int64
	DeleteCount int64
}

func (s *DeleteBytesSplice) Splice() {}

func (s *DeleteBytesSplice) SpliceEqual(b EditFileSplice) bool {
	s2, ok := b.(*DeleteBytesSplice)
	if !ok {
		return false
	}
	return s.Index == s2.Index && s.DeleteCount == s2.DeleteCount
}

func (s *DeleteBytesSplice) String() string {
	return fmt.Sprintf("Delete(%d, %d)", s.Index, s.DeleteCount)
}

type InsertBytesFileOp struct {
	Path  string
	Index int64
	Data  Bytes
}

func (op *InsertBytesFileOp) FilePath() string { return op.Path }

func (op *InsertBytesFileOp) Op() {}

func (op *InsertBytesFileOp) OpEqual(b Op) bool {
	op2, ok := b.(*InsertBytesFileOp)
	if !ok {
		return false
	}
	return op.Path == op2.Path && op.Data.Equal(op2.Data) && op.Index == op2.Index
}

func (op *InsertBytesFileOp) String() string {
	// To make debugging easier, we print the string if it's short.
	if op.Data.Len() < 20 {
		return fmt.Sprintf("InsertBytesFileOp(%s, %d, %q)", op.Path, op.Index, op.Data.String())
	} else {
		return fmt.Sprintf("InsertBytesFileOp(%s, %d, bytes[%d])", op.Path, op.Index, op.Data.Len())
	}
}

type DeleteBytesFileOp struct {
	Path        string
	Index       int64
	DeleteCount int64
}

func (op *DeleteBytesFileOp) FilePath() string { return op.Path }

func (op *DeleteBytesFileOp) Op() {}

func (op *DeleteBytesFileOp) OpEqual(b Op) bool {
	op2, ok := b.(*DeleteBytesFileOp)

	if !ok {
		return false
	}

	return op.Path == op2.Path && op.Index == op2.Index && op.DeleteCount == op2.DeleteCount
}

func (op *DeleteBytesFileOp) String() string {
	return fmt.Sprintf("DeleteBytesFileOp(%s, %d, %d)", op.Path, op.Index, op.DeleteCount)
}

type ChmodFileOp struct {
	Path       string
	Executable bool
}

func (op *ChmodFileOp) FilePath() string { return op.Path }

func (op *ChmodFileOp) Op() {}

func (op *ChmodFileOp) OpEqual(b Op) bool {
	op2, ok := b.(*ChmodFileOp)

	if !ok {
		return false
	}

	if op.FilePath() != op2.FilePath() || op.Executable != op2.Executable {
		return false
	}

	return true
}

func (op *ChmodFileOp) String() string {
	return fmt.Sprintf("ChmodFileOp(%s, %v)", op.Path, op.Executable)
}

// ClearOp is an Operation that clears the entire tree of the Snapshot; its result is always the
// empty Snapshot
type ClearOp struct {
}

func (op *ClearOp) Op() {}

func (op *ClearOp) OpEqual(b Op) bool {
	if _, ok := b.(*ClearOp); ok {
		return true
	}

	return false

}

func (op *ClearOp) String() string {
	return fmt.Sprintf("ClearOp")
}

// RmdirOp deletes all files under Path
type RmdirOp struct {
	Path string
}

func (op *RmdirOp) Op() {}

func (op *RmdirOp) OpEqual(b Op) bool {
	op2, ok := b.(*RmdirOp)

	if !ok {
		return false
	}

	return op.Path == op2.Path
}

func (op *RmdirOp) String() string {
	return fmt.Sprintf("RmdirOp(%s)", op.Path)
}

type SubdirOp struct {
	Path string
}

func (op *SubdirOp) Op() {}

func (op *SubdirOp) OpEqual(b Op) bool {
	op2, ok := b.(*SubdirOp)

	if !ok {
		return false
	}

	return op.Path == op2.Path
}

func (op *SubdirOp) String() string {
	return fmt.Sprintf("SubdirOp(%s)", op.Path)
}

type DirOp struct {
	Names []string
}

func (op *DirOp) Op() {}

func (op *DirOp) OpEqual(b Op) bool {
	op2, ok := b.(*DirOp)
	if !ok {
		return false
	}
	if len(op.Names) != len(op2.Names) {
		return false
	}
	for i, n := range op.Names {
		if n != op2.Names[i] {
			return false
		}
	}
	return true
}

func (op *DirOp) String() string {
	return fmt.Sprintf("DirOp(%s)", strings.Join(op.Names, ","))
}

type PreserveOp struct {
	Matcher       *dbpath.Matcher
	StripContents bool
}

func (op *PreserveOp) Op() {}

func (op *PreserveOp) OpEqual(b Op) bool {
	op2, ok := b.(*PreserveOp)
	if !ok {
		return false
	}
	return op.Matcher.Equal(op2.Matcher) && op.StripContents == op2.StripContents
}

func (op *PreserveOp) String() string {
	return fmt.Sprintf("PreserveOp(%s,%v)", strings.Join(op.Matcher.ToPatterns(), ","), op.StripContents)
}

type IdentityOp struct{}

func (op *IdentityOp) Op() {}

func (op *IdentityOp) OpEqual(b Op) bool {
	if _, ok := b.(*IdentityOp); ok {
		return true
	}

	return false
}

func (op *IdentityOp) String() string {
	return "IdentityOp"
}

// Overlay two or more snapshots on top of each other.
// Intended to replicate the semantics of overlayfs with multple lower dirs.
// https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/tree/Documentation/filesystems/overlayfs.txt
type OverlayOp struct{}

func (op *OverlayOp) Op() {}

func (op *OverlayOp) OpEqual(b Op) bool {
	if _, ok := b.(*OverlayOp); ok {
		return true
	}

	return false
}

func (op *OverlayOp) String() string {
	return "OverlayOp"
}

// An op for internal synchronization of channels. Not intended for permanent storage,
// and we will bail out if you try to convert it to a protobuf.
type SyncOp struct {
	Token string
}

func (op *SyncOp) Op() {}

func (op *SyncOp) OpEqual(b Op) bool {
	op2, ok := b.(*SyncOp)

	if !ok {
		return false
	}

	return op.Token == op2.Token
}

func (op *SyncOp) String() string {
	return fmt.Sprintf("SyncOp(%s)", op.Token)
}

// An op purely for simulating playback failure.
// Evaluators should always interpret this as an error.
type FailureOp struct {
	Message string
}

func (op *FailureOp) Op() {}

func (op *FailureOp) OpEqual(b Op) bool {
	op2, ok := b.(*FailureOp)

	if !ok {
		return false
	}

	return op.Message == op2.Message
}

func (op *FailureOp) String() string {
	return fmt.Sprintf("FailureOp(%q)", op.Message)
}
