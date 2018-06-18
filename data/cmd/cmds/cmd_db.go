package cmds

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/golang/protobuf/proto"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/cmd"
	cmdProto "github.com/windmilleng/mish/data/cmd/proto"
	"github.com/windmilleng/mish/data/db/db2"
	"github.com/windmilleng/mish/data/db/storage"
	dataProto "github.com/windmilleng/mish/data/proto"
	"github.com/windmilleng/mish/data/transform"
	ospathConv "github.com/windmilleng/mish/os/ospath/convert"
)

// CmdDB is able to find an existing Run we can use.
// It doesn't run the command, it just allows the client to set an ID
// and will dutifully return it.
type CmdDB struct {
	recipes storage.AllRecipeReader
	store   cmd.RunStore
}

func NewCmdDB(recipes storage.AllRecipeReader, store cmd.RunStore) *CmdDB {
	return &CmdDB{recipes: recipes, store: store}
}

// For the command, set the RunID if it's currently empty.
// If s is the empty string, it won't set it, and it's just a get.
func (db *CmdDB) GetAssociatedID(ctx context.Context, c cmd.Cmd) (cmd.AssociatedID, []data.Op, error) {
	return db.get(ctx, nil, c)
}

const maxDepth = 10

func (db *CmdDB) get(ctx context.Context, ops []data.Op, c cmd.Cmd) (cmd.AssociatedID, []data.Op, error) {
	if len(ops) > maxDepth {
		return "", nil, nil
	}
	r, changed, err := db.useRunIfPossible(ctx, c, ops)
	bestChanged := changed

	// If we can't use this run because there are Ops that preclude it,
	// don't bother returning.
	if r != "" || len(changed) > 0 || err != nil {
		return r, changed, err
	}

	srs, err := db.recipes.AllPathsToSnapshot(ctx, c.Dir)
	if err != nil {
		return "", nil, err
	}

	for _, sr := range srs {
		op := sr.Op
		if !db2.FriendIsLinearOp(op) || len(sr.Recipe.Inputs) != 1 {
			continue
		}

		srOps := ops
		if _, ok := op.(*data.IdentityOp); !ok {
			srOps = append(srOps, op)
		}

		srC := c
		srC.Dir = sr.Recipe.Inputs[0]

		r, changed, err := db.get(ctx, srOps, srC)
		// we want the shortest non-zero explanation
		if len(bestChanged) == 0 || (len(changed) > 0 && len(changed) < len(bestChanged)) {
			bestChanged = changed
		}
		if r != "" || err != nil {
			return r, changed, err
		}
	}

	return "", bestChanged, nil
}

// returns:
// *) AssociatedID, if we can use it
// *) invalidated ops if we can't (or empty if we can't find a related run)
// *) error
func (db *CmdDB) useRunIfPossible(ctx context.Context, c cmd.Cmd, ops []data.Op) (cmd.AssociatedID, []data.Op, error) {
	key, err := db.Hash(ctx, c)
	if err != nil {
		return "", nil, err
	}

	info, err := db.store.GetRunInfo(ctx, key)
	if err != nil {
		return "", nil, err
	}

	if c.T == cmd.PlainCmdType {
		if len(ops) > 0 {
			return "", ops, nil
		}
	} else {
		if len(ops) == 0 {
			return info.ID, nil, nil
		}
		// can we transform?
		if info.Matcher == nil {
			// we don't have anything to preserve against, so no
			return "", nil, nil
		}
	}

	if info.ID == "" {
		return "", nil, nil
	}

	pOp := &data.PreserveOp{Matcher: info.Matcher}

	for _, op := range ops {
		op, err := transform.LeftEclipsingTransform(op, pOp)
		if err != nil {
			return "", []data.Op{op}, err
		}

		if _, ok := op.(*data.IdentityOp); !ok {
			return "", []data.Op{op}, nil
		}
	}

	return info.ID, nil, nil
}

func (db *CmdDB) Hash(ctx context.Context, c cmd.Cmd) (cmd.Key, error) {
	var m *dataProto.Matcher
	if c.SnapshotFS != nil {
		m = ospathConv.MatcherD2P(c.SnapshotFS)
	}

	var t cmdProto.Command_CmdType
	switch c.T {
	case cmd.PlainCmdType:
		t = cmdProto.Command_PLAIN
	case cmd.PreserveCmdType:
		t = cmdProto.Command_PRESERVE
	case cmd.PluginCmdType:
		t = cmdProto.Command_PLUGIN
	default:
		return "", fmt.Errorf("unknown CmdType %v", t)
	}

	p := &cmdProto.Command{
		Dir:           c.Dir.String(),
		Argv:          c.Argv,
		SnapshotFs:    m,
		CmdType:       t,
		DepsFile:      c.DepsFile,
		Env:           c.Env,
		Owner:         string(c.Owner),
		ContainerName: string(c.ContainerName),
		// We explicitly exclude Artifacts. Artifacts are an optimization,
		// so they shouldn't rule out a run
		// Artifact:   artifacts,
	}

	bytes, err := proto.Marshal(p)
	if err != nil {
		return "", err
	}

	hash := sha256.New()
	if _, err = hash.Write(bytes); err != nil {
		return "", err
	}

	bytes = hash.Sum(nil)

	// Use base64 encoding so that we can represent this as a UTF-8 string
	bytesEncoded := base64.URLEncoding.EncodeToString(bytes)
	return cmd.Key(bytesEncoded), nil
}
