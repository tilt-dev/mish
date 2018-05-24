package dbint

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbpath"
)

// An interface for reading files off a known snapshot.
//
// This API is designed to make it easier to batch snapshot queries.
//
// A naive function would take a snapshot ID and kick off a "cold" query:
// func LookupResult(id snapID) { return d.SnapshotFile(ctx, id, "exitcode") }
//
// But if we take a SnapshotFileReader as an input, the caller can be more clever about
// batching requests and passing in a "warmed up" reader.
// func LookupResult(r SnapshotFileReader) { return r.File(ctx, "exitcode") }
type SnapshotFileReader interface {
	// Returns the state of a file at this snapshot.
	// Returns a grpc NotFound if the file cannot be found.
	//
	// TODO(nick): Separate out corruption errors (playback failure)
	// from transient errors (DB failure)
	File(ctx context.Context, path string) (*SnapshotFile, error)

	// Returns the state of all matching files.
	//
	// SnapshotFileList is itself a SnapshotFileReader, so can be chained.
	FilesMatching(ctx context.Context, matcher *dbpath.Matcher) (SnapshotFileList, error)
}

type dbSnapshotFileReader struct {
	Reader
	data.SnapshotID
}

func (r *dbSnapshotFileReader) File(ctx context.Context, path string) (*SnapshotFile, error) {
	return r.Reader.SnapshotFile(ctx, r.SnapshotID, path)
}

func (r *dbSnapshotFileReader) FilesMatching(ctx context.Context, matcher *dbpath.Matcher) (SnapshotFileList, error) {
	return r.Reader.SnapshotFilesMatching(ctx, r.SnapshotID, matcher)
}

func FileReaderForSnapshot(db Reader, id data.SnapshotID) SnapshotFileReader {
	return &dbSnapshotFileReader{Reader: db, SnapshotID: id}
}

type SnapshotMetadata interface {
	ID() data.SnapshotID

	ContainsPath(path string) bool

	PathSet() map[string]bool
}

type SnapshotFileList []*SnapshotFile

func (l SnapshotFileList) File(ctx context.Context, path string) (*SnapshotFile, error) {
	for _, f := range l {
		if f.Path == path {
			return f, nil
		}
	}
	return nil, grpc.Errorf(codes.NotFound, "File not found %q", path)
}

func (l SnapshotFileList) FilesMatching(ctx context.Context, matcher *dbpath.Matcher) (SnapshotFileList, error) {
	result := make(SnapshotFileList, 0)
	for _, f := range l {
		if matcher.Match(f.Path) {
			result = append(result, f)
		}
	}
	return result, nil
}

func (l SnapshotFileList) AsMap() map[string]*SnapshotFile {
	result := make(map[string]*SnapshotFile, len(l))
	for _, f := range l {
		result[f.Path] = f
	}
	return result
}

// Clients of this package should treat this struct as immutable.
type SnapshotFile struct {
	Path       string
	Contents   data.Bytes
	Executable bool
	Type       data.FileType
}

type SnapshotCost struct {
	Ops int32

	// Excludes ops that produce an optimized snapshot.
	NonOptimizedOps int32
}
