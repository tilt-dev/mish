package cmd

import (
	"context"
	"fmt"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/dbpath"
	"github.com/windmilleng/mish/os/ospath"
	"github.com/windmilleng/mish/runner"
)

type CmdType int

const (
	PlainCmdType CmdType = iota
	PreserveCmdType
	PluginCmdType
)

type Cmd struct {
	Dir           data.SnapshotID
	Argv          []string
	SnapshotFS    *ospath.Matcher
	T             CmdType
	DepsFile      string
	Env           map[string]string
	Owner         data.UserID
	Artifacts     runner.ArtifactRequest
	ContainerName runner.ContainerName
	Network       bool
}

func (c Cmd) String() string {
	var m string
	if c.SnapshotFS != nil {
		m = c.SnapshotFS.String()
	}
	return fmt.Sprintf("Cmd{Dir: %q, Argv: %q, SnapshotFS: %v, T: %v, DepsFile: %q, Env: %v, Owner: %v, Artifacts: %v, ContainerName: %q, Network: %v}",
		c.Dir,
		c.Argv,
		m,
		c.T,
		c.DepsFile,
		c.Env,
		c.Owner,
		c.Artifacts,
		c.ContainerName,
		c.Network,
	)
}

// The value to associate with a Cmd
// for now, should be a Composer.ID, but we don't want to make Cmd depend on Composer
type AssociatedID string

type CmdDB interface {
	// Returns the Associated ID, or a list of Ops that negated, or an error
	// or all zero values if we couldn't find a related run
	GetAssociatedID(ctx context.Context, c Cmd) (AssociatedID, []data.Op, error)

	Hash(ctx context.Context, c Cmd) (Key, error)
}

type Key string

type RunInfo struct {
	ID      AssociatedID
	Matcher *dbpath.Matcher
}

type RunStore interface {
	GetRunInfo(ctx context.Context, key Key) (RunInfo, error)
}
