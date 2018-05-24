package proto

import (
	"fmt"

	"github.com/windmilleng/mish/data"
)

func WriteP2D(p *Write) (data.Write, error) {
	if p == nil {
		return nil, fmt.Errorf("WriteP2D: nil write")
	}

	switch p := p.W.(type) {
	case *Write_WCreateSnapshot:
		w := p.WCreateSnapshot
		r, err := StoredRecipeP2D(w.StoredRecipe)
		if err != nil {
			return nil, err
		}
		return data.CreateSnapshotWrite{Recipe: r}, nil
	case *Write_WSetPointer:
		next, err := PointerAtSnapshotP2D(p.WSetPointer.Next)
		if err != nil {
			return nil, err
		}
		return data.SetPointerWrite{Next: next}, nil
	case *Write_WAcquirePointer:
		id, err := data.ParsePointerID(p.WAcquirePointer.Id)
		if err != nil {
			return nil, err
		}
		host := data.Host(p.WAcquirePointer.Host)
		return data.AcquirePointerWrite{ID: id, Host: host}, nil
	default:
		return nil, fmt.Errorf("WriteP2D: unexpected write type %T %v", p, p)
	}
}

func WritesP2D(ps []*Write) (ds []data.Write, err error) {
	for _, p := range ps {
		d, err := WriteP2D(p)
		if err != nil {
			return nil, err
		}
		ds = append(ds, d)
	}
	return ds, nil
}

func WriteD2P(d data.Write) (p *Write, err error) {
	p = &Write{}
	switch d := d.(type) {
	case data.CreateSnapshotWrite:
		recipeP, err := StoredRecipeD2P(d.Recipe)
		if err != nil {
			return nil, err
		}
		return &Write{W: &Write_WCreateSnapshot{&CreateSnapshotWrite{
			StoredRecipe: recipeP,
		}}}, nil
	case data.SetPointerWrite:
		next := PointerAtSnapshotD2P(d.Next)
		return &Write{W: &Write_WSetPointer{&SetPointerWrite{Next: next}}}, nil
	case data.AcquirePointerWrite:
		id := d.ID.String()
		host := string(d.Host)
		return &Write{W: &Write_WAcquirePointer{&AcquirePointerWrite{Id: id, Host: host}}}, nil
	default:
		return nil, fmt.Errorf("WriteD2P: unexpected write type %T %v", d, d)
	}
}

func WritesD2P(ds []data.Write) (ps []*Write, err error) {
	for _, d := range ds {
		p, err := WriteD2P(d)
		if err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	return ps, nil
}
