package users

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/markbates/goth"
	"github.com/windmilleng/mish/data"
)

var gothUserNick = goth.User{
	UserID:   "123",
	Provider: "github",
	Name:     "nicks",
	RawData: map[string]interface{}{
		"login": "nicks",
	},
}

var gothUserDan = goth.User{
	UserID:   "1234",
	Provider: "github",
	Name:     "dbentley",
	RawData: map[string]interface{}{
		"login": "dbentley",
	},
}

func TestMemoryStore(t *testing.T) {
	f := newMemoryFixture(t)
	f.TestStore()
}

func TestMemoryWhitelist(t *testing.T) {
	f := newMemoryFixture(t)
	f.TestWhitelist()
}

func TestMemoryGothUserConflict(t *testing.T) {
	f := newMemoryFixture(t)
	f.TestGothUserConflict()
}

func TestMemoryUsernameConflict(t *testing.T) {
	f := newMemoryFixture(t)
	f.TestUsernameConflict()
}

func TestMemoryUsernameConflictCase(t *testing.T) {
	f := newMemoryFixture(t)
	f.TestUsernameConflictCase()
}

func TestNativeAccounts(t *testing.T) {
	f := newMemoryFixture(t)
	f.TestNativeAccounts()
}

func TestMemoryDeleteUser(t *testing.T) {
	f := newMemoryFixture(t)
	f.TestDeleteUser()
}

func TestMemorySetPublic(t *testing.T) {
	f := newMemoryFixture(t)
	f.TestSetPublic()
}

type fixture struct {
	t     *testing.T
	ctx   context.Context
	store data.UserStore
}

func newMemoryFixture(t *testing.T) fixture {
	ctx := context.Background()
	store, err := NewMemoryUserStore(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return fixture{t: t, ctx: ctx, store: store}
}

func (f fixture) TestStore() {
	t := f.t
	s := f.store
	ctx := f.ctx

	gothUser := gothUserNick
	username := data.Username("nick")

	_, err := s.LookupByUsername(ctx, username)
	if err == nil || grpc.Code(err) != codes.NotFound {
		t.Errorf("Expected NotFound error. Actual: %v", err)
	}

	_, err = s.LookupByGothUser(ctx, gothUser)
	if err == nil || grpc.Code(err) != codes.NotFound {
		t.Errorf("Expected NotFound error. Actual: %v", err)
	}

	user, err := s.CreateFromGothUser(ctx, username, gothUser)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.LookupByUsername(ctx, username)
	if err != nil {
		t.Errorf("Missing user by username: %s", username)
	}

	_, err = s.LookupByGothUser(ctx, gothUser)
	if err != nil {
		t.Errorf("Missing user by goth user: %v", gothUser)
	}

	user2, err := s.LookupByUserID(ctx, user.UserID)
	if err != nil {
		t.Errorf("Missing user by user id: %d", user.UserID)
	}

	if user != user2 {
		t.Errorf("Users different: (%+v, %+v)", user, user2)
	}

	err = s.DeleteByUserID(ctx, user.UserID)

	if err != nil {
		t.Errorf("Error deleting user: %v", err)
	}

	_, err = s.LookupByUsername(ctx, username)
	if err == nil || grpc.Code(err) != codes.NotFound {
		t.Errorf("Expected NotFound error. Actual: %v", err)
	}

	user, err = s.CreateFromGothUser(ctx, username, gothUser)
	if err != nil {
		t.Fatal(err)
	}
}

func (f fixture) TestWhitelist() {
	t := f.t
	s := f.store
	ctx := f.ctx

	gothUser := gothUserNick
	gothUser.RawData = map[string]interface{}{"login": "Ab"}
	username := data.Username("nick")

	_, err := s.CreateFromGothUser(ctx, username, gothUser)
	if err == nil || grpc.Code(err) != codes.PermissionDenied {
		t.Errorf("Expected PermissionDenied error. Actual: %v", err)
	}

	logins, err := s.WhitelistedGithubLogins(ctx)
	if err != nil {
		t.Fatal(err)
	} else if len(logins) != len(defaultGithubLogins) {
		t.Errorf("Expected %d whitelisted logins. Actual: %+v", len(defaultGithubLogins), logins)
	}

	err = s.WhitelistGithubLogin(ctx, "aB")
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.CreateFromGothUser(ctx, username, gothUser)
	if err != nil {
		t.Fatal(err)
	}

	logins, err = s.WhitelistedGithubLogins(ctx)
	if err != nil {
		t.Fatal(err)
	} else if len(logins) != len(defaultGithubLogins)+1 {
		t.Errorf("Expected %d whitelisted logins. Actual: %+v", len(defaultGithubLogins)+1, logins)
	}
}

func (f fixture) TestGothUserConflict() {
	t := f.t
	s := f.store
	ctx := f.ctx
	gothUser := gothUserNick
	username := data.Username("nick")
	username2 := data.Username("nick2")

	_, err := s.CreateFromGothUser(ctx, username, gothUser)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.CreateFromGothUser(ctx, username2, gothUser)
	if err == nil || grpc.Code(err) != codes.AlreadyExists {
		t.Errorf("Expected user conflict error. Actual: %v", err)
	}
}

func (f fixture) TestUsernameConflict() {
	t := f.t
	s := f.store
	ctx := f.ctx

	gothUser := gothUserNick
	gothUser2 := gothUserDan
	username := data.Username("nick")

	_, err := s.CreateFromGothUser(ctx, username, gothUser)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.CreateFromGothUser(ctx, username, gothUser2)
	if err == nil || grpc.Code(err) != codes.AlreadyExists {
		t.Errorf("Expected user conflict error. Actual: %v", err)
	}
}

func (f fixture) TestUsernameConflictCase() {
	t := f.t
	s := f.store
	ctx := f.ctx

	gothUser := gothUserNick
	gothUser2 := gothUserDan
	username1 := data.Username("NIck")
	username2 := data.Username("niCK")

	_, err := s.CreateFromGothUser(ctx, username1, gothUser)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.CreateFromGothUser(ctx, username2, gothUser2)
	if err == nil || grpc.Code(err) != codes.AlreadyExists {
		t.Errorf("Expected user conflict error. Actual: %v", err)
	}
}

func (f fixture) TestNativeAccounts() {
	t := f.t
	s := f.store
	ctx := f.ctx

	root, err := s.LookupByUsername(ctx, data.Root)
	if err != nil {
		t.Fatal(err)
	} else if root.UserID != data.RootID {
		t.Errorf("Unexpected user id: %x", root.UserID)
	}

	anon, err := s.LookupByUsername(ctx, data.Anonymous)
	if err != nil {
		t.Fatal(err)
	} else if anon.UserID != data.AnonymousID {
		t.Errorf("Unexpected user id: %x", anon.UserID)
	}

	public, err := s.LookupByUsername(ctx, data.Public)
	if err != nil {
		t.Fatal(err)
	} else if public.UserID != data.PublicID {
		t.Errorf("Unexpected user id: %x", public.UserID)
	}
}

func (f fixture) TestDeleteUser() {
	t := f.t
	ctx := f.ctx
	s := f.store
	gothUser1 := gothUserNick
	username1 := data.Username("nick")
	gothUser2 := gothUserDan
	username2 := data.Username("dmill")

	user1, err := s.CreateFromGothUser(ctx, username1, gothUser1)
	if err != nil {
		t.Fatal(err)
	}

	user2, err := s.CreateFromGothUser(ctx, username2, gothUser2)
	if err != nil {
		t.Fatal(err)
	}

	err = s.DeleteByUserID(ctx, user1.UserID)
	if err != nil {
		t.Errorf("Error deleting user: %v", err)
	}

	_, err = s.LookupByUsername(ctx, user2.Username)
	if err != nil {
		t.Fatal(err)
	}
}

func (f fixture) TestSetPublic() {
	t := f.t
	ctx := f.ctx
	s := f.store
	gothUser := gothUserNick
	username := data.Username("nick")

	user, err := s.CreateFromGothUser(ctx, username, gothUser)
	if err != nil {
		t.Fatal(err)
	}

	err = s.SetPublicForDemos(ctx, user.UserID, true)
	if err != nil {
		t.Fatal(err)
	}

	user, err = s.LookupByUserID(ctx, user.UserID)
	if err != nil {
		t.Fatal(err)
	} else if !user.IsPublic {
		t.Errorf("Expected public user")
	}

	err = s.SetPublicForDemos(ctx, user.UserID, false)
	if err != nil {
		t.Fatal(err)
	}

	user, err = s.LookupByUserID(ctx, user.UserID)
	if err != nil {
		t.Fatal(err)
	} else if user.IsPublic {
		t.Errorf("Expected non-public user")
	}
}
