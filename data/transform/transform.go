package transform

import (
	"fmt"
	"path"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbpath"
)

var noTransform = fmt.Errorf("No transform available")

func IsNoTransformErr(err error) bool {
	return err == noTransform
}

// The fundamental operational transform.
//
// Defined as the function f
// f(A, B) = (A', B')
// where B' ∘ A = A' ∘ B,
//
// Less formally, this means that if we have any snapshot,
// adding A then B' on top of that snapshot is equivalent
// to adding B then A' on top of that snapshot.
//
// TODO(nick): Traditional operational transform equations assume
// a linear sequence of ops (i.e., each function A() takes exactly one
// argument). But our ops might take more than one argument. We haven't
// rigorously defined what this means for the definition of OT.
//
// I *think* what we need to do is define OT as
// f(A, iA, B, iB) = (A', B')
// where B' ∘[iB] A = A' ∘[iA] B,
// defining A ∘[x] B as the composition of A with B as the xth argument
// to A, but maybe there's some prior art here that can help us
// nail down the notation.
func Transform(a, b data.Op) (data.Op, data.Op, error) {
	// The IdentityOp commutes with every other op, so do that first.
	_, isAIdentity := a.(*data.IdentityOp)
	if isAIdentity {
		return a, b, nil
	}

	_, isBIdentity := b.(*data.IdentityOp)
	if isBIdentity {
		return a, b, nil
	}

	aDir, isADir := a.(*data.DirOp)
	if isADir {
		bPrime := opVsDir(b, aDir)
		if bPrime == nil {
			return nil, nil, noTransform
		}
		return a, bPrime, nil
	}

	bDir, isBDir := b.(*data.DirOp)
	if isBDir {
		aPrime := opVsDir(a, bDir)
		if aPrime == nil {
			return nil, nil, noTransform
		}
		return aPrime, b, nil
	}

	aPreserve, isAPreserve := a.(*data.PreserveOp)
	bPreserve, isBPreserve := b.(*data.PreserveOp)
	if isAPreserve && isBPreserve {
		return preserveVsPreserve(aPreserve, bPreserve)
	}

	if isAPreserve {
		bPrime := opVsPreserve(b, aPreserve)
		if bPrime == nil {
			return nil, nil, noTransform
		}
		return a, bPrime, nil
	}

	if isBPreserve {
		aPrime := opVsPreserve(a, bPreserve)
		if aPrime == nil {
			return nil, nil, noTransform
		}
		return aPrime, b, nil
	}

	return nil, nil, noTransform
}

func opVsPreserve(op data.Op, p *data.PreserveOp) (r data.Op) {
	fOp, ok := op.(data.FileOp)
	if !ok {
		return nil
	}

	if p.StripContents {
		if _, ok := fOp.(*data.EditFileOp); ok {
			return &data.IdentityOp{}
		}
		if _, ok := fOp.(*data.InsertBytesFileOp); ok {
			return &data.IdentityOp{}
		}
		if _, ok := fOp.(*data.DeleteBytesFileOp); ok {
			return &data.IdentityOp{}
		}
		if wOp, ok := fOp.(*data.WriteFileOp); ok {
			copy := *wOp
			copy.Data = data.NewEmptyBytes()
			return &copy
		}
	}

	if !p.Matcher.Match(fOp.FilePath()) {
		return &data.IdentityOp{}
	}

	return op
}

func preserveVsPreserve(a, b *data.PreserveOp) (data.Op, data.Op, error) {
	aPaths := a.Matcher.AsFileSet()
	if aPaths != nil {
		aPrime := pathsVsPreserve(aPaths, b)
		if aPrime == nil {
			return nil, nil, noTransform
		}
		return aPrime, b, nil
	}

	bPaths := b.Matcher.AsFileSet()
	if bPaths != nil {
		bPrime := pathsVsPreserve(bPaths, a)
		if bPrime == nil {
			return nil, nil, noTransform
		}
		return a, bPrime, nil
	}

	return nil, nil, noTransform
}

// Right now, we only know how to merge to preserves when one is a file whitelist.
func pathsVsPreserve(paths []string, p *data.PreserveOp) (r data.Op) {
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		if p.Matcher.Match(path) {
			result = append(result, path)
		}
	}
	newMatcher, err := dbpath.NewFilesMatcher(result)
	if err != nil {
		// It would be really weird if this happened, because all these
		// paths come from another matcher.
		panic(err)
	}
	return &data.PreserveOp{Matcher: newMatcher}
}

func opVsDir(op data.Op, d *data.DirOp) (r data.Op) {
	if len(d.Names) != 1 {
		// See comments on Transform() about non-linear ops.
		return nil
	}

	switch op := op.(type) {
	case *data.EditFileOp:
		copy := *op
		copy.Path = path.Join(d.Names[0], op.Path)
		return &copy
	case *data.InsertBytesFileOp:
		copy := *op
		copy.Path = path.Join(d.Names[0], op.Path)
		return &copy
	case *data.DeleteBytesFileOp:
		copy := *op
		copy.Path = path.Join(d.Names[0], op.Path)
		return &copy
	case *data.WriteFileOp:
		copy := *op
		copy.Path = path.Join(d.Names[0], op.Path)
		return &copy
	case *data.RemoveFileOp:
		copy := *op
		copy.Path = path.Join(d.Names[0], op.Path)
		return &copy
	case *data.PreserveOp:
		return &data.PreserveOp{Matcher: op.Matcher.SubdirDB(d.Names[0])}
	}

	return nil
}

// An eclipsing transform is when the transform is invariant over one argument.
//
// Defined when
// f(A, B) = (A', B)
// where B ∘ A = A' ∘ B
//
// LeftEclipsing just means that the right argument eclipses the left (b is invariant).
// Returns A' if eclipsing, nil if this is non-eclipsing.
func LeftEclipsingTransform(a, b data.Op) (data.Op, error) {
	aPrime, bPrime, err := Transform(a, b)
	if err != nil {
		if IsNoTransformErr(err) {
			return nil, nil
		}
		return nil, err
	}

	if bPrime != b {
		return nil, nil
	}
	return aPrime, nil
}

// A commutative transform is when
// B ∘ A = A ∘ B
func IsCommutative(a, b data.Op) bool {
	aPrime, bPrime, err := Transform(a, b)
	if err != nil {
		return false
	}
	return aPrime == a && bPrime == b
}
