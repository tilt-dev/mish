package data

type Consistency int32

// The request will take a lot longer but will have no false negatives.
const FullConsistency Consistency = 1

// For when it's ok if we have false negatives
// (e.g., we have the snapshot in the DB, but HasSnapshots doesn't return it)
const FromCache Consistency = 2
