package data

import (
	"testing"

	"github.com/windmilleng/mish/data/db/dbpath"
)

func TestOpEquals(t *testing.T) {
	p1 := "p1"
	p2 := "p2"
	b1 := BytesFromString("b1")
	b2 := BytesFromString("b2")
	s1 := &InsertBytesSplice{Index: 0, Data: b1}
	s2 := &InsertBytesSplice{Index: 0, Data: b2}
	s3 := &DeleteBytesSplice{Index: 0, DeleteCount: 1}
	m1, _ := dbpath.NewMatcherFromPattern("foo")
	m2, _ := dbpath.NewMatcherFromPattern("bar")

	// The test assumes that all ops in this slice are different.
	ops := []Op{
		&WriteFileOp{Path: p1, Data: b1},
		&WriteFileOp{Path: p1, Data: b2},
		&WriteFileOp{Path: p2, Data: b2},
		&RemoveFileOp{Path: p1},
		&RemoveFileOp{Path: p2},
		&EditFileOp{Path: p1, Splices: []EditFileSplice{s1}},
		&EditFileOp{Path: p1, Splices: []EditFileSplice{s2}},
		&EditFileOp{Path: p1, Splices: []EditFileSplice{s3}},
		&EditFileOp{Path: p1, Splices: []EditFileSplice{s1, s2}},
		&InsertBytesFileOp{Path: p1, Data: b1},
		&InsertBytesFileOp{Path: p1, Data: b2},
		&DeleteBytesFileOp{Path: p1, DeleteCount: 1},
		&DeleteBytesFileOp{Path: p1, DeleteCount: 2},
		&ChmodFileOp{Path: p1, Executable: false},
		&ChmodFileOp{Path: p1, Executable: true},
		&ClearOp{},
		&RmdirOp{Path: p1},
		&RmdirOp{Path: p2},
		&SubdirOp{Path: p1},
		&SubdirOp{Path: p2},
		&DirOp{Names: []string{"a", "b"}},
		&PreserveOp{Matcher: m1},
		&PreserveOp{Matcher: m2},
		&IdentityOp{},
		&OverlayOp{},
		&SyncOp{Token: "a"},
		&SyncOp{Token: "b"},
		&FailureOp{Message: "x"},
	}

	l := len(ops)
	for i := 0; i < l; i++ {
		for j := 0; j < l; j++ {
			isEqual := ops[i].OpEqual(ops[j])
			expected := i == j
			if isEqual && !expected {
				t.Errorf("Expected ops to be different: (%s, %s)", ops[i], ops[j])
			} else if !isEqual && expected {
				t.Errorf("Expected ops to be the same: (%s, %s)", ops[i], ops[j])
			}
		}
	}
}

func TestDifferentInstancesOfOpsWithSameData(t *testing.T) {
	p1 := "p1"
	p2 := "p1"
	b1 := BytesFromString("b1")
	b2 := BytesFromString("b1")

	s1 := &InsertBytesSplice{Index: 0, Data: b1}
	s2 := &InsertBytesSplice{Index: 0, Data: b2}

	m1, _ := dbpath.NewMatcherFromPattern("foo")
	m2, _ := dbpath.NewMatcherFromPattern("foo")

	// these two ops should be the same
	var sameOps = []struct {
		first  Op
		second Op
	}{
		{&EditFileOp{Path: p1, Splices: []EditFileSplice{s1}}, &EditFileOp{Path: p1, Splices: []EditFileSplice{s2}}},
		{
			&EditFileOp{Path: p1, Splices: []EditFileSplice{&DeleteBytesSplice{Index: 0, DeleteCount: 2}}},
			&EditFileOp{Path: p2, Splices: []EditFileSplice{&DeleteBytesSplice{Index: 0, DeleteCount: 2}}},
		},
		{&WriteFileOp{Path: p1, Data: b1}, &WriteFileOp{Path: p2, Data: b2}},
		{&RemoveFileOp{Path: p1}, &RemoveFileOp{Path: p2}},
		{&InsertBytesFileOp{Path: p1, Data: b1}, &InsertBytesFileOp{Path: p2, Data: b2}},
		{&DeleteBytesFileOp{Path: p1, DeleteCount: 1}, &DeleteBytesFileOp{Path: p2, DeleteCount: 1}},
		{&ChmodFileOp{Path: p1, Executable: false}, &ChmodFileOp{Path: p2, Executable: false}},
		{&ClearOp{}, &ClearOp{}},
		{&RmdirOp{Path: p1}, &RmdirOp{Path: p2}},
		{&SubdirOp{Path: p1}, &SubdirOp{Path: p2}},
		{&DirOp{Names: []string{"a", "b"}}, &DirOp{Names: []string{"a", "b"}}},
		{&PreserveOp{Matcher: m1}, &PreserveOp{Matcher: m2}},
		{&IdentityOp{}, &IdentityOp{}},
		{&OverlayOp{}, &OverlayOp{}},
		{&SyncOp{Token: "a"}, &SyncOp{Token: "a"}},
		{&FailureOp{Message: "x"}, &FailureOp{Message: "x"}},
	}

	for _, tt := range sameOps {
		isEqual := tt.first.OpEqual(tt.second)
		if !isEqual {
			t.Errorf("Expected ops to be the same: (%s, %s)", tt.first, tt.second)
		}
	}
}
