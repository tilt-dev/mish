package data

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/windmilleng/mish/logging"
)

// SnapshotID identifies an immutable Snapshot.
// TODO(nick): SnapshotID is in a transition state from a string to more rich struct.
type SnapshotID struct {
	t SnapshotType

	// The owner of this snapshot id
	ownerID UserID

	// All snapshot creators (wmdaemon, wmapi) require a nonce
	// so that they don't conflict.
	sourceNonce ClientNonce

	// All snapshot creators (wmdaemon, wmapi) assign a unique index
	// to each snapshot from that creator, hex-encoded.
	sourceIndex string

	// ContentID-based and Temp snapshots have a hash that uniquely identify it.
	hash string
}

type SnapshotType string

const (
	SnapshotTemp     SnapshotType = "temp"
	SnapshotTest                  = "test"
	SnapshotDaemon                = "wmdaemon"
	SnapshotAPI                   = "wmapi"
	SnapshotRunner                = "wmrunner"
	SnapshotStorage               = "wmstorage"
	SnapshotFrontend              = "wmfrontend"
	SnapshotCLI                   = "wmcli"
	SnapshotEmpty                 = "empty"
	SnapshotSHA                   = "sha256"
)

func ParseSnapshotID(s string) SnapshotID {
	ownerID := AnonymousID
	uIndex := strings.Index(s, "$")
	if uIndex != -1 {
		ownerStr := s[:uIndex]
		ownerInt, err := strconv.ParseUint(ownerStr, 16, 64)
		if err != nil {
			logging.Global().Errorf("Malformed snapshot id: %q\n", s)
			return SnapshotID{}
		}
		ownerID = UserID(ownerInt)
		s = s[uIndex+1:]
	}
	return ParseLocalSnapshotID(s, ownerID)
}

func ParseLocalSnapshotID(s string, ownerID UserID) SnapshotID {
	if s == "" {
		return SnapshotID{}
	}

	if s == SnapshotEmpty {
		return EmptySnapshotID
	}

	parts := strings.SplitN(s, "-", 3)
	t := SnapshotType(parts[0])
	if t == SnapshotTemp || t == SnapshotSHA || t == SnapshotTest {
		if len(parts) != 2 {
			logging.Global().Errorf("Malformed snapshot id: %q\n", s)
			return SnapshotID{}
		}
		return SnapshotID{ownerID: ownerID, t: t, hash: parts[1]}
	}

	if t == SnapshotDaemon || t == SnapshotAPI || t == SnapshotFrontend || t == SnapshotStorage || t == SnapshotRunner || t == SnapshotCLI {
		if len(parts) != 3 {
			logging.Global().Errorf("Malformed snapshot id: %q\n", s)
			return SnapshotID{}
		}
		return SnapshotID{ownerID: ownerID, t: t, sourceNonce: ClientNonce(parts[1]), sourceIndex: parts[2]}
	}

	logging.Global().Errorf("Malformed snapshot id: %q\n", s)
	return SnapshotID{}
}

func (id SnapshotID) Nil() bool {
	return id.t == ""
}

func (id SnapshotID) Owner() UserID {
	return id.ownerID
}

// The local name of a snapshot, without the owner ID.
// We generally want to use this for content-based hashing, so that the owner ID
// doesn't get mixed in to the hash.
func (id SnapshotID) Local() string {
	s := id.String()
	uIndex := strings.Index(s, "$")
	if uIndex == -1 {
		return s
	} else {
		return s[uIndex+1:]
	}
}

func (id SnapshotID) String() string {
	if id.Nil() {
		return ""
	} else if id.t == SnapshotEmpty {
		return string(id.t)
	} else if id.t == SnapshotTemp || id.t == SnapshotSHA || id.t == SnapshotTest {
		return fmt.Sprintf("%x$%s-%s", id.ownerID, id.t, id.hash)
	}
	return fmt.Sprintf("%x$%s-%s-%s", id.ownerID, id.t, id.sourceNonce, id.sourceIndex)
}

func (id SnapshotID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.String())
}

func (id SnapshotID) IsContentID() bool {
	return id.t == SnapshotEmpty || id.t == SnapshotSHA
}

func (id SnapshotID) IsEmptyID() bool {
	return id.t == SnapshotEmpty
}

func (id SnapshotID) Hash() ([]byte, error) {
	if id.t == SnapshotEmpty {
		return nil, nil
	} else if id.t == SnapshotSHA {
		return hex.DecodeString(id.hash)
	}
	return nil, fmt.Errorf("Snapshot does not have a hash: %s", id)
}

// SnapshotPrefix allows you to define incrementing snapshots.
type SnapshotPrefix struct {
	t SnapshotType

	nonce ClientNonce
}

func NewSpokeSnapshotPrefix(t ClientType, nonce ClientNonce) SnapshotPrefix {
	return SnapshotPrefix{t: t.SnapshotType(), nonce: nonce}
}

func NewTestSnapshotPrefix() SnapshotPrefix {
	return SnapshotPrefix{t: SnapshotTest}
}

func (p SnapshotPrefix) NewID(ownerID UserID, i int64) SnapshotID {
	if p.t == SnapshotTest {
		return OwnedTestID(ownerID, i)
	} else {
		return SnapshotID{
			t:           p.t,
			ownerID:     ownerID,
			sourceNonce: p.nonce,
			sourceIndex: fmt.Sprintf("%x", i),
		}
	}
}

// Special SnapshotID's
var EmptySnapshotID SnapshotID = SnapshotID{t: SnapshotEmpty}

// ID of any temporary, transient snapshot.
// Caches and storage should refuse to index any snapshot with this ID.
func TempID(t string) SnapshotID {
	return SnapshotID{t: SnapshotTemp, ownerID: AnonymousID, hash: t}
}

func (id SnapshotID) IsTempID() bool {
	return id.t == SnapshotTemp
}

// ID for testing.
func TestID(index int64) SnapshotID {
	return OwnedTestID(AnonymousID, index)
}

func OwnedTestID(ownerID UserID, index int64) SnapshotID {
	return SnapshotID{t: SnapshotTest, ownerID: ownerID, hash: fmt.Sprintf("%x", index)}
}

// ID based on the checksum hash of a recipe.
func ContentID(ownerID UserID, hash []byte) SnapshotID {
	return SnapshotID{t: SnapshotSHA, ownerID: ownerID, hash: fmt.Sprintf("%x", hash)}
}

type TempWriteError struct {
	ID SnapshotID
}

func (e TempWriteError) Error() string {
	return fmt.Sprintf("Can't write temp ID %v", e.ID)
}

// An Op describes an Operation to create a Snapshot
type Op interface {
	fmt.Stringer

	// No-op method to limit what's considered an Op
	Op()

	OpEqual(op2 Op) bool
}

type FileOp interface {
	Op

	FilePath() string
}

// Recipe describes how to create a new Snapshot using an Operation and Input
type Recipe struct {
	// Inputs to Op; for now must be either 0 or 1
	Inputs []SnapshotID
	Op     Op
}

type StoredRecipe struct {
	Snap SnapshotID
	Tag  RecipeWTag
	Recipe
}

type TaggedRecipe struct {
	Tag RecipeWTag
	Recipe
}

func ReverseStoredRecipes(recipes []StoredRecipe) {
	l := len(recipes)
	for i, _ := range recipes {
		j := l - 1 - i
		if i >= j {
			return
		}
		recipes[i], recipes[j] = recipes[j], recipes[i]
	}
}

func ReverseOps(ops []Op) {
	l := len(ops)
	for i, _ := range ops {
		j := l - 1 - i
		if i >= j {
			return
		}
		ops[i], ops[j] = ops[j], ops[i]
	}
}

type RecipeTagType int

const (
	// Follows the path of recipes that a user actually edited.
	RecipeTagTypeEdit RecipeTagType = iota

	// Follows the path of recipes for optimal lookup.
	RecipeTagTypeOptimal

	// Temporary recipes for scanning purposes. These are never replicated
	// across servers.
	RecipeTagTypeTemp

	// Follows the path of recipes for optimal reuse
	RecipeTagTypeRewritten
)

// When we write a Recipe to the DB, we attach a write tag.
// Write tags must be replicated exactly on database copy.
type RecipeWTag struct {
	Type RecipeTagType

	// The Tag ID. Semantics depend on the type.
	// For Edit tags, this is the pointer ID
	// For Rewritten tags, it is horizontal or vertical
	ID string
}

// A snapshot in the DB may have multiple recipe paths.
//
// Read tags use fuzzy matching to select a path, so the recipe that comes out
// might not have the exact same tag as we asked for.
type RecipeRTag struct {
	Type RecipeTagType

	// The Tag ID. Semantics depend on the type. For EDIT tags, this is the pointer ID
	ID string
}

// For when we want a recipe optimized for reading, rather than
// reflecting the original edit history.
var RecipeRTagOptimal = RecipeRTag{Type: RecipeTagTypeOptimal}
var RecipeWTagOptimal = RecipeWTag{Type: RecipeTagTypeOptimal}

// For when we want an edit recipe, but we don't care which one.
var RecipeRTagEdit = RecipeRTag{Type: RecipeTagTypeEdit}
var RecipeWTagEdit = RecipeWTag{Type: RecipeTagTypeEdit}

// For when we want a recipe that was not put in by the caller,
// but determined to be equivalent and better
var RecipeRTagRewritten = RecipeRTag{Type: RecipeTagTypeRewritten}
var RecipeWTagRewritten = RecipeWTag{Type: RecipeTagTypeRewritten}

const RewriteIDBuildDirection = "buildDirection"
const RewriteIDEditDirection = "editDirection"

// Used for bookkeeping during directory scanning.
// Never used for lookups, because there's no good reason
// to care what order the scanner discovered files in.
var RecipeWTagTemp = RecipeWTag{Type: RecipeTagTypeTemp}

// For when we want an edit recipe along the path of a particular pointer.
func RecipeRTagForPointer(id PointerID) RecipeRTag {
	return RecipeRTag{Type: RecipeTagTypeEdit, ID: id.String()}
}
func RecipeWTagForPointer(id PointerID) RecipeWTag {
	return RecipeWTag{Type: RecipeTagTypeEdit, ID: id.String()}
}

type Pointers interface {
	// Makes a Temporary Pointer at Rev 1 -> EmptySnapshot
	// Prefix should be a string to identify the pointer. E.g. "wm-init-workflow"
	// or "dbentley" or some such.
	MakeTemp(c context.Context, userID UserID, prefix string, t PointerType) (PointerAtSnapshot, error)

	// Acquire a pointer for writing.
	// If the pointer doesn't exist, create it.
	//
	// Unlike hub#AcquirePointer, this doesn't accept a Host, because we expect
	// the write Host to be implied by the machine we're currently on.
	AcquirePointer(c context.Context, id PointerID) (PointerAtRev, error)

	// Read the current revision of a pointer. It is ok if the pointer is a bit stale.
	// If the pointer doesn't exist, return an error.
	Head(c context.Context, id PointerID) (PointerAtRev, error)

	// Get returns the value of a Pointer at a revision
	Get(c context.Context, v PointerAtRev) (PointerAtSnapshot, error)

	// Set sets a the Pointer to next iff it is a valid next revision
	// (incremented Rev, increasing UpdatedAt, current head is not frozen)
	Set(c context.Context, next PointerAtSnapshot) error

	// Wait waits for a Pointer to change from a known value or ctx to be done (returning nil)
	Wait(c context.Context, last PointerAtRev) error

	// Returns all pointer IDs for this user
	ActivePointerIDs(c context.Context, userID UserID, types []PointerType) ([]PointerID, error)
}

type PointerListener interface {
	OnPointerUpdate(c context.Context, newHead PointerAtSnapshot) error
}
