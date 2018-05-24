package watch

import "github.com/windmilleng/mish/data"

type WatchEvent interface {
	isWatchEvent()
}

// New ops have been created
type WatchOpsEvent struct {
	SnapID data.SnapshotID
	Ops    []data.Op
}

func (e WatchOpsEvent) isWatchEvent() {}

func (e WatchOpsEvent) IsEmpty() bool {
	return len(e.Ops) == 0
}

// There was an error.
type WatchErrEvent struct {
	Err error
}

func (e WatchErrEvent) isWatchEvent() {}

// Somebody requested an fsync.
type WatchSyncEvent struct {
	Token string
}

func (e WatchSyncEvent) isWatchEvent() {}
