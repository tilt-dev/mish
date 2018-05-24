package proto

import (
	"encoding/json"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/golang/protobuf/jsonpb"
	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbpath"
)

func RecipeP2D(p *Recipe) (r data.Recipe, err error) {
	if p == nil {
		return r, fmt.Errorf("RecipeP2D: nil recipe")
	}
	for _, i := range p.InputSnapId {
		r.Inputs = append(r.Inputs, data.ParseSnapshotID(i))
	}

	switch op := p.Op.(type) {
	case *Recipe_OpWriteFile:
		w := op.OpWriteFile
		r.Op = &data.WriteFileOp{
			Path:       pathP2D(w.Path),
			Data:       data.NewBytesWithBacking(w.Data),
			Executable: w.Executable,
			Type:       data.FileType(w.Type),
		}
	case *Recipe_OpEditFile:
		w := op.OpEditFile
		r.Op = &data.EditFileOp{
			Path:    pathP2D(w.Path),
			Splices: splicesP2D(w.Splices),
		}
	case *Recipe_OpRemoveFile:
		w := op.OpRemoveFile
		r.Op = &data.RemoveFileOp{
			Path: pathP2D(w.Path),
		}
	case *Recipe_OpInsertBytesFile:
		w := op.OpInsertBytesFile
		r.Op = &data.InsertBytesFileOp{
			Path:  pathP2D(w.Path),
			Index: w.Index,
			Data:  data.NewBytesWithBacking(w.Data),
		}
	case *Recipe_OpDeleteBytesFile:
		w := op.OpDeleteBytesFile
		r.Op = &data.DeleteBytesFileOp{
			Path:        pathP2D(w.Path),
			Index:       w.Index,
			DeleteCount: w.DeleteCount,
		}
	case *Recipe_OpChmodFile:
		w := op.OpChmodFile
		r.Op = &data.ChmodFileOp{
			Path:       pathP2D(w.Path),
			Executable: w.Executable,
		}
	case *Recipe_OpRmdir:
		w := op.OpRmdir
		r.Op = &data.RmdirOp{
			Path: pathP2D(w.Path),
		}
	case *Recipe_OpSubdir:
		w := op.OpSubdir
		r.Op = &data.SubdirOp{
			Path: pathP2D(w.Path),
		}
	case *Recipe_OpDir:
		w := op.OpDir
		names := make([]string, len(w.Name))
		for i, name := range w.Name {
			names[i] = pathP2D(name)
		}
		r.Op = &data.DirOp{
			Names: names,
		}
	case *Recipe_OpPreserve:
		w := op.OpPreserve
		m, err := dbpath.NewMatcherFromPatterns(w.Patterns)
		if err != nil {
			return data.Recipe{}, err
		}
		r.Op = &data.PreserveOp{
			Matcher:       m,
			StripContents: w.StripContents,
		}
	case *Recipe_OpIdentity:
		r.Op = &data.IdentityOp{}
	case *Recipe_OpOverlay:
		r.Op = &data.OverlayOp{}
	case *Recipe_OpFailure:
		r.Op = &data.FailureOp{Message: op.OpFailure.Msg}
	default:
		return r, fmt.Errorf("RecipeP2D: unexpected op type %T %v in %v", op, op, r)
	}

	return r, nil
}

func RecipesP2D(ps []*Recipe) (ds []data.Recipe, err error) {
	for _, p := range ps {
		d, err := RecipeP2D(p)
		if err != nil {
			return nil, err
		}
		ds = append(ds, d)
	}
	return ds, nil
}

func StoredRecipeP2D(p *StoredRecipe) (data.StoredRecipe, error) {
	tag := RecipeWTagP2D(p.Tag)

	d, err := RecipeP2D(p.Recipe)
	if err != nil {
		return data.StoredRecipe{}, err
	}

	return data.StoredRecipe{Snap: data.ParseSnapshotID(p.SnapId), Recipe: d, Tag: tag}, nil
}

func StoredRecipesP2D(ps []*StoredRecipe) ([]data.StoredRecipe, error) {
	result := make([]data.StoredRecipe, len(ps))
	for i, p := range ps {
		r, err := StoredRecipeP2D(p)
		if err != nil {
			return nil, err
		}
		result[i] = r
	}
	return result, nil
}

func RecipeD2P(d data.Recipe) (p *Recipe, err error) {
	inputs := []string{}
	for _, i := range d.Inputs {
		inputs = append(inputs, i.String())
	}

	switch op := d.Op.(type) {
	case *data.WriteFileOp:
		return &Recipe{
			InputSnapId: inputs,
			Op: &Recipe_OpWriteFile{OpWriteFile: &WriteFileOp{
				Path:       pathD2P(op.Path),
				Data:       op.Data.InternalByteSlice(),
				Executable: op.Executable,
				Type:       FileType(op.Type),
			}},
		}, nil
	case *data.EditFileOp:
		return &Recipe{
			InputSnapId: inputs,
			Op: &Recipe_OpEditFile{OpEditFile: &EditFileOp{
				Path:    pathD2P(op.Path),
				Splices: splicesD2P(op.Splices),
			}},
		}, nil
	case *data.RemoveFileOp:
		return &Recipe{
			InputSnapId: inputs,
			Op: &Recipe_OpRemoveFile{OpRemoveFile: &RemoveFileOp{
				Path: pathD2P(op.Path),
			}},
		}, nil
	case *data.ChmodFileOp:
		return &Recipe{
			InputSnapId: inputs,
			Op: &Recipe_OpChmodFile{OpChmodFile: &ChmodFileOp{
				Path:       pathD2P(op.Path),
				Executable: op.Executable,
			}},
		}, nil
	case *data.InsertBytesFileOp:
		return &Recipe{
			InputSnapId: inputs,
			Op: &Recipe_OpInsertBytesFile{OpInsertBytesFile: &InsertBytesFileOp{
				Path:  pathD2P(op.Path),
				Index: op.Index,
				Data:  op.Data.InternalByteSlice(),
			}},
		}, nil
	case *data.DeleteBytesFileOp:
		return &Recipe{
			InputSnapId: inputs,
			Op: &Recipe_OpDeleteBytesFile{OpDeleteBytesFile: &DeleteBytesFileOp{
				Path:        pathD2P(op.Path),
				Index:       op.Index,
				DeleteCount: op.DeleteCount,
			}},
		}, nil
	case *data.RmdirOp:
		return &Recipe{
			InputSnapId: inputs,
			Op: &Recipe_OpRmdir{OpRmdir: &RmdirOp{
				Path: pathD2P(op.Path),
			}},
		}, nil
	case *data.SubdirOp:
		return &Recipe{
			InputSnapId: inputs,
			Op: &Recipe_OpSubdir{OpSubdir: &SubdirOp{
				Path: pathD2P(op.Path),
			}},
		}, nil
	case *data.DirOp:
		names := make([]*Path, len(op.Names))
		for i, name := range op.Names {
			names[i] = pathD2P(name)
		}
		return &Recipe{
			InputSnapId: inputs,
			Op: &Recipe_OpDir{OpDir: &DirOp{
				Name: names,
			}},
		}, nil
	case *data.PreserveOp:
		return &Recipe{
			InputSnapId: inputs,
			Op: &Recipe_OpPreserve{OpPreserve: &PreserveOp{
				Patterns:      op.Matcher.ToPatterns(),
				StripContents: op.StripContents,
			}},
		}, nil
	case *data.IdentityOp:
		return &Recipe{
			InputSnapId: inputs,
			Op:          &Recipe_OpIdentity{OpIdentity: &IdentityOp{}},
		}, nil
	case *data.OverlayOp:
		return &Recipe{
			InputSnapId: inputs,
			Op:          &Recipe_OpOverlay{OpOverlay: &OverlayOp{}},
		}, nil
	case *data.FailureOp:
		return &Recipe{
			InputSnapId: inputs,
			Op:          &Recipe_OpFailure{OpFailure: &FailureOp{Msg: op.Message}},
		}, nil
	default:
		return nil, fmt.Errorf("RecipeD2P: unexpected op type %T %v in %v", op, op, d)
	}
}

func RecipesD2P(ds []data.Recipe) (ps []*Recipe, err error) {
	for _, d := range ds {
		p, err := RecipeD2P(d)
		if err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	return ps, nil
}

func StoredRecipeD2P(d data.StoredRecipe) (*StoredRecipe, error) {
	tag := RecipeWTagD2P(d.Tag)
	p, err := RecipeD2P(d.Recipe)
	if err != nil {
		return nil, err
	}

	return &StoredRecipe{SnapId: d.Snap.String(), Recipe: p, Tag: tag}, nil
}

func StoredRecipesD2P(rs []data.StoredRecipe) ([]*StoredRecipe, error) {
	result := make([]*StoredRecipe, len(rs))
	for i, r := range rs {
		p, err := StoredRecipeD2P(r)
		if err != nil {
			return nil, err
		}
		result[i] = p
	}
	return result, nil
}

func PointerAtRevP2D(p *PointerAtRev) (r data.PointerAtRev, err error) {
	if p == nil {
		return r, fmt.Errorf("PointerAtRevP2D: nil p")
	}
	id, err := data.ParsePointerID(p.Id)
	if err != nil {
		return r, err
	}
	r.ID = id
	r.Rev = data.PointerRev(p.Rev)
	return r, nil
}

func PointerAtRevsP2D(ps []*PointerAtRev) ([]data.PointerAtRev, error) {
	result := make([]data.PointerAtRev, len(ps))
	var err error
	for i, p := range ps {
		result[i], err = PointerAtRevP2D(p)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func PointerAtRevD2P(d data.PointerAtRev) *PointerAtRev {
	return &PointerAtRev{
		Id:  d.ID.String(),
		Rev: int64(d.Rev),
	}
}

func PointerAtRevsD2P(ds []data.PointerAtRev) ([]*PointerAtRev, error) {
	result := make([]*PointerAtRev, len(ds))
	for i, d := range ds {
		result[i] = PointerAtRevD2P(d)
	}
	return result, nil
}

func PointerAtSnapshotP2D(p *PointerAtSnapshot) (data.PointerAtSnapshot, error) {
	var t time.Time
	if p.UpdatedAtNs != 0 {
		t = time.Unix(0, p.UpdatedAtNs)
	}
	id, err := data.ParsePointerID(p.Id)
	if err != nil {
		return data.PointerAtSnapshot{}, err
	}
	return data.PointerAtSnapshot{
		ID:        id,
		Rev:       data.PointerRev(p.Rev),
		SnapID:    data.ParseSnapshotID(p.SnapId),
		Frozen:    p.Frozen,
		UpdatedAt: t,
	}, nil
}

func PointerAtSnapshotD2P(d data.PointerAtSnapshot) *PointerAtSnapshot {
	updatedAtNS := d.UpdatedAt.UnixNano()
	if d.UpdatedAt.IsZero() {
		updatedAtNS = 0
	}
	return &PointerAtSnapshot{
		Id:          d.ID.String(),
		Rev:         int64(d.Rev),
		SnapId:      d.SnapID.String(),
		Frozen:      d.Frozen,
		UpdatedAtNs: updatedAtNS,
	}
}

func PointerAtSnapshotsP2D(p []*PointerAtSnapshot) ([]data.PointerAtSnapshot, error) {
	result := make([]data.PointerAtSnapshot, 0, len(p))
	for _, el := range p {
		s, err := PointerAtSnapshotP2D(el)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}

func PointerAtSnapshotsD2P(d []data.PointerAtSnapshot) []*PointerAtSnapshot {
	result := make([]*PointerAtSnapshot, 0, len(d))
	for _, el := range d {
		result = append(result, PointerAtSnapshotD2P(el))
	}
	return result
}

func pathD2P(path string) *Path {
	if utf8.ValidString(path) {
		return &Path{Val: &Path_Utf8{Utf8: path}}
	} else {
		return &Path{Val: &Path_Raw{Raw: []byte(path)}}
	}
}

func pathP2D(path *Path) string {
	switch val := path.Val.(type) {
	case (*Path_Utf8):
		return val.Utf8
	case (*Path_Raw):
		return string(val.Raw)
	default:
		return ""
	}
}

// Custom JSONPB conversion for paths.
func (p *Path) MarshalJSONPB(m *jsonpb.Marshaler) ([]byte, error) {
	switch val := p.Val.(type) {
	case (*Path_Utf8):
		return json.Marshal(val.Utf8)
	case (*Path_Raw):
		s, err := json.Marshal(val.Raw)
		if err != nil {
			return nil, err
		}
		return []byte(fmt.Sprintf(`{"raw":%s}`, s)), nil
	default:
		return []byte("{}"), nil
	}
}

func (p *Path) UnmarshalJSONPB(m *jsonpb.Unmarshaler, b []byte) error {
	if len(b) == 0 {
		return fmt.Errorf("Malformed Path")
	}

	firstChar := string(b[0])
	if firstChar == `"` {
		utf8 := &Path_Utf8{}
		p.Val = utf8
		return json.Unmarshal(b, &(utf8.Utf8))
	}
	raw := &Path_Raw{}
	p.Val = raw
	return json.Unmarshal(b, raw)
}

func RecipeTagTypeP2D(t RecipeTagType) data.RecipeTagType {
	switch t {
	case RecipeTagType_EDIT:
		return data.RecipeTagTypeEdit
	case RecipeTagType_OPTIMAL:
		return data.RecipeTagTypeOptimal
	case RecipeTagType_REWRITTEN:
		return data.RecipeTagTypeRewritten
	default:
		return data.RecipeTagTypeEdit
	}
}

func RecipeTagTypeD2P(t data.RecipeTagType) RecipeTagType {
	switch t {
	case data.RecipeTagTypeEdit:
		return RecipeTagType_EDIT
	case data.RecipeTagTypeOptimal:
		return RecipeTagType_OPTIMAL
	case data.RecipeTagTypeRewritten:
		return RecipeTagType_REWRITTEN
	default:
		return RecipeTagType_EDIT
	}
}

func RecipeRTagP2D(t *RecipeRTag) data.RecipeRTag {
	if t == nil {
		return data.RecipeRTagEdit
	}
	return data.RecipeRTag{
		Type: RecipeTagTypeP2D(t.Type),
		ID:   t.Id,
	}
}

func RecipeRTagD2P(t data.RecipeRTag) *RecipeRTag {
	return &RecipeRTag{
		Type: RecipeTagTypeD2P(t.Type),
		Id:   t.ID,
	}
}

func RecipeWTagP2D(t *RecipeWTag) data.RecipeWTag {
	if t == nil {
		return data.RecipeWTagEdit
	}
	return data.RecipeWTag{
		Type: RecipeTagTypeP2D(t.Type),
		ID:   t.Id,
	}
}

func RecipeWTagD2P(t data.RecipeWTag) *RecipeWTag {
	return &RecipeWTag{
		Type: RecipeTagTypeD2P(t.Type),
		Id:   t.ID,
	}
}

func splicesP2D(splices []*EditFileSplice) []data.EditFileSplice {
	result := make([]data.EditFileSplice, len(splices))
	for i, s := range splices {
		result[i] = spliceP2D(s)
	}
	return result
}

func splicesD2P(splices []data.EditFileSplice) []*EditFileSplice {
	result := make([]*EditFileSplice, len(splices))
	for i, s := range splices {
		result[i] = spliceD2P(s)
	}
	return result
}

func spliceP2D(splice *EditFileSplice) data.EditFileSplice {
	switch edit := splice.Edit.(type) {
	case *EditFileSplice_DeleteCount:
		return &data.DeleteBytesSplice{
			Index:       splice.Index,
			DeleteCount: edit.DeleteCount,
		}
	case *EditFileSplice_Data:
		return &data.InsertBytesSplice{
			Index: splice.Index,
			Data:  data.NewBytesWithBacking(edit.Data),
		}
	default:
		panic(fmt.Sprintf("Unrecognized splice type %T", edit))
	}
}

func spliceD2P(splice data.EditFileSplice) *EditFileSplice {
	switch splice := splice.(type) {
	case *data.InsertBytesSplice:
		return &EditFileSplice{
			Index: splice.Index,
			Edit: &EditFileSplice_Data{
				Data: splice.Data.InternalByteSlice(),
			},
		}
	case *data.DeleteBytesSplice:
		return &EditFileSplice{
			Index: splice.Index,
			Edit: &EditFileSplice_DeleteCount{
				DeleteCount: splice.DeleteCount,
			},
		}
	default:
		panic(fmt.Sprintf("Unrecognized splice type %T", splice))
	}
}

func PointersP2D(ptrs []string) ([]data.PointerID, error) {
	return data.StringsToPointerIDs(ptrs)
}

func PointersD2P(ptrs []data.PointerID) []string {
	return data.PointerIDsToStrings(ptrs)
}

func SnapshotIDsP2D(snaps []string) []data.SnapshotID {
	result := make([]data.SnapshotID, len(snaps))
	for i, s := range snaps {
		result[i] = data.ParseSnapshotID(s)
	}
	return result
}

func SnapshotIDsD2P(snaps []data.SnapshotID) []string {
	result := make([]string, len(snaps))
	for i, s := range snaps {
		result[i] = s.String()
	}
	return result
}

func ConsistencyD2P(c data.Consistency) Consistency {
	switch c {
	case data.FullConsistency:
		return Consistency_FULL_CONSISTENCY
	case data.FromCache:
		return Consistency_FROM_CACHE
	default:
		// If the value is unrecognized, default to full consistency for forwardcompat
		return Consistency_FULL_CONSISTENCY
	}
}

func ConsistencyP2D(c Consistency) data.Consistency {
	switch c {
	case Consistency_FULL_CONSISTENCY:
		return data.FullConsistency
	case Consistency_FROM_CACHE:
		return data.FromCache
	default:
		// If the value is unrecognized, default to full consistency for forwardcompat
		return data.FullConsistency
	}
}

func PointerMetadataP2D(m *PointerMetadata) data.PointerMetadata {
	return data.PointerMetadata{WriteHost: data.Host(m.WriteHost)}
}

func PointerMetadataD2P(m data.PointerMetadata) *PointerMetadata {
	return &PointerMetadata{WriteHost: string(m.WriteHost)}
}
