package dbint

import (
	"context"
	"sort"

	"github.com/windmilleng/mish/data"
)

// A helper class for writing directories of files, which is a common operation
// when writing Windmill snapshots (most notably used when optimizing a snapshot
// for reads).
//
// Aggregates errors until it's time to Commit()
type DirAssembler struct {
	ctx    context.Context
	db     DB2
	owner  data.UserID
	tag    data.RecipeWTag
	err    error
	names  []string
	inputs []data.SnapshotID
}

// ctx: The context for all writes
// db: The database to write into
func NewDirAssembler(ctx context.Context, db DB2, owner data.UserID, tag data.RecipeWTag) *DirAssembler {
	return &DirAssembler{
		ctx:    ctx,
		db:     db,
		tag:    tag,
		err:    nil,
		owner:  owner,
		names:  make([]string, 0),
		inputs: make([]data.SnapshotID, 0),
	}
}

func (da *DirAssembler) WriteString(name string, contents string) {
	da.Write(name, data.BytesFromString(contents), false, data.FileRegular)
}

func (da *DirAssembler) Write(name string, contents data.Bytes, executable bool, fileType data.FileType) {
	if da.err != nil {
		return
	}

	id, _, err := da.db.Create(da.ctx, data.Recipe{
		Op: &data.WriteFileOp{
			Path:       "",
			Data:       contents,
			Executable: executable,
			Type:       fileType,
		},
	}, da.owner, da.tag)
	if err != nil {
		da.err = err
		return
	}

	da.WriteSnapshot(name, id)
}

func (da *DirAssembler) WriteSnapshot(name string, snap data.SnapshotID) {
	da.names = append(da.names, name)
	da.inputs = append(da.inputs, snap)
}

func (da *DirAssembler) Commit() (data.SnapshotID, error) {
	recipe, err := da.AsRecipe()
	if err != nil {
		return data.SnapshotID{}, err
	}
	id, _, err := da.db.Create(da.ctx, recipe, da.owner, da.tag)
	return id, err
}

func (da *DirAssembler) AsRecipe() (data.Recipe, error) {
	if da.err != nil {
		return data.Recipe{}, da.err
	}

	// Always sort the names, so the DirOp is in content-addressable form.
	sortable := &dirOpSortable{
		names:  da.names,
		inputs: da.inputs,
	}
	sort.Sort(sortable)
	return data.Recipe{Op: &data.DirOp{Names: sortable.names}, Inputs: sortable.inputs}, nil
}

// A struct for sorting dir ops.
type dirOpSortable struct {
	inputs []data.SnapshotID
	names  []string
}

func (d *dirOpSortable) Len() int           { return len(d.inputs) }
func (d *dirOpSortable) Less(i, j int) bool { return d.names[i] < d.names[j] }
func (d *dirOpSortable) Swap(i, j int) {
	d.inputs[i], d.inputs[j] = d.inputs[j], d.inputs[i]
	d.names[i], d.names[j] = d.names[j], d.names[i]
}
