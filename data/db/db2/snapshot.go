package db2

import (
	"context"
	"fmt"
	"strings"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbint"
	"github.com/windmilleng/mish/data/db/dbpath"
	"github.com/windmilleng/mish/tracing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// Evaluate the snapshot at the given file path. If no path is specified, we
// evaluate the whole snapshot.
func (s *DB2) evalSnapshotAtPath(ctx context.Context, snap data.SnapshotID, matcher *dbpath.Matcher, stripContents bool) (*db2Snapshot, error) {
	span, ctx := tracing.StartSystemSpanFromContext(ctx, "db2/evalSnapshotAtPath", opentracing.Tags{
		"patterns":            strings.Join(matcher.ToPatterns(), ","),
		"snap":                snap,
		string(ext.Component): "db2",
	})
	defer span.Finish()

	result, err := s.eval(ctx, data.RecipeRTagOptimal, snap, &snapshotEvaluator{matcher: matcher, stripContents: stripContents})
	if err != nil {
		return nil, err
	}

	switch result := result.(type) {
	case *snapshotDir:
		return &db2Snapshot{snap, result, pathSet(result)}, nil
	case *snapshotFile:
		dir := newDir()
		dir.files["."] = result
		return &db2Snapshot{snap, dir, pathSet(dir)}, nil
	default:
		return nil, fmt.Errorf("eval: unexpected opEvalResult type %T %v", result, result)
	}
}

func (s *DB2) Snapshot(ctx context.Context, snap data.SnapshotID) (dbint.SnapshotMetadata, error) {
	return s.evalSnapshotAtPath(ctx, snap, dbpath.NewAllMatcher(), true)
}

func (s *DB2) SnapshotFile(ctx context.Context, snap data.SnapshotID, path string) (*dbint.SnapshotFile, error) {
	matcher, err := dbpath.NewFileMatcher(path)
	if err != nil {
		return nil, err
	}

	v, err := s.evalSnapshotAtPath(ctx, snap, matcher, false)
	if err != nil {
		return nil, err
	}

	file, err := v.File(path)
	if err != nil {
		return nil, err
	}

	if file == nil {
		return nil, grpc.Errorf(codes.NotFound, "File not found %q", path)
	}
	return file, nil
}

func (s *DB2) SnapshotFilesMatching(ctx context.Context, snap data.SnapshotID, matcher *dbpath.Matcher) (dbint.SnapshotFileList, error) {
	v, err := s.evalSnapshotAtPath(ctx, snap, matcher, false)
	if err != nil {
		return nil, err
	}

	return v.FilesMatching(matcher)
}

type db2Snapshot struct {
	id      data.SnapshotID
	d       *snapshotDir
	pathSet map[string]bool
}

func (s *db2Snapshot) ID() data.SnapshotID {
	return s.id
}

func (s *db2Snapshot) ContainsPath(path string) bool {
	f, _ := lookupFile(s.d, path, onNotFoundDoNothing)
	return f != nil
}

func (s *db2Snapshot) PathSet() map[string]bool {
	result := make(map[string]bool)
	for k, v := range s.pathSet {
		result[k] = v
	}
	return result
}

func (s *db2Snapshot) File(path string) (*dbint.SnapshotFile, error) {
	f, _ := lookupFile(s.d, path, onNotFoundDoNothing)
	if f == nil {
		return nil, grpc.Errorf(codes.NotFound, "File not found %q", path)
	}

	return &dbint.SnapshotFile{
		Path:       path,
		Contents:   f.data,
		Executable: f.executable,
		Type:       f.fileType,
	}, nil
}

func (s *db2Snapshot) FilesMatching(matcher *dbpath.Matcher) (dbint.SnapshotFileList, error) {
	results := make(dbint.SnapshotFileList, 0)
	for k, _ := range s.pathSet {
		if matcher.Match(k) {
			f, err := s.File(k)
			if err != nil {
				return nil, err
			}
			results = append(results, f)
		}
	}
	return results, nil
}
