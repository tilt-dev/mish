package data

import (
	"context"
	"regexp"
	"strings"
	"unicode"

	"github.com/markbates/goth"
)

type UserReader interface {
	LookupByUserID(ctx context.Context, userID UserID) (User, error)
}

type UserStore interface {
	UserReader

	DeleteByUserID(ctx context.Context, userID UserID) error

	LookupByUsername(ctx context.Context, username Username) (User, error)

	LookupByGothUser(ctx context.Context, user goth.User) (User, error)

	CreateFromGothUser(ctx context.Context, username Username, user goth.User) (User, error)

	WhitelistGithubLogin(ctx context.Context, login string) error

	WhitelistedGithubLogins(ctx context.Context) ([]string, error)

	SetPublicForDemos(ctx context.Context, userID UserID, public bool) error
}

type Authenticator interface {
	CurrentUser(ctx context.Context) (User, error)
}

type UserID uint64
type Username string

var usernameRE = regexp.MustCompile("^[a-z][a-z0-9_]{2,}$")

// Creates a new username, trimming and normalizing.
// Returns the empty username if the username is not valid, or if it's a resrved name.
func NewUsername(str string) Username {
	username := NormalizeUsername(Username(str))

	// Forbid reserved usernames
	if username == Anonymous || username == Root {
		return ""
	}

	return username
}

// Returns the empty username if the username is not valid.
func NormalizeUsername(u Username) Username {
	str := string(u)
	str = strings.TrimSpace(str)
	str = strings.ToLower(str)
	if !usernameRE.MatchString(str) {
		return ""
	}
	return Username(str)
}

func (n Username) Empty() bool {
	return n == Anonymous || n == ""
}

type User struct {
	UserID      UserID
	Username    Username
	Email       string
	AvatarURL   string
	Name        string
	FirstName   string
	LastName    string
	Description string
	Location    string

	// If true, we disable all ACL checks for this user's data
	IsPublic bool
}

func (u User) Nil() bool {
	return u.UserID == 0
}

// Used for cases where we don't need to assign an owner
// (like temp data that is never written to the database, or test data)
const Anonymous Username = "nym"
const AnonymousID UserID = 2

// The root user. Should only be used for system processes that need
// access to all pointers.
const Root Username = "root"
const RootID UserID = 1

// A public user. All users can read this data. No users can write this data.
// Should only be written by internal system calls.
const Public Username = "public"
const PublicID UserID = 1000

// Intended to be temporary, where we need a user but don't have one yet.
const UserTODO = Anonymous
const UserTODOID = AnonymousID

// For testing
const UserTest = Anonymous
const UserTestID = AnonymousID

func IsValidGithubLogin(s string) bool {
	return s != "" && strings.IndexFunc(s, unicode.IsSpace) == -1
}
