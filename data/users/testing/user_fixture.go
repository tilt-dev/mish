package testing

import (
	"context"
	"fmt"
	"testing"

	"github.com/markbates/goth"
	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/users"
	"github.com/windmilleng/mish/net/auth"
)

type UserFixture struct {
	Users data.UserStore
	UserA data.User
	UserB data.User
	CtxA  context.Context
	CtxB  context.Context

	ctx       context.Context
	t         *testing.T
	lastIndex uint
}

func NewUserFixture(ctx context.Context, t *testing.T) *UserFixture {
	users, err := users.NewMemoryUserStore(ctx)
	if err != nil {
		t.Fatal(err)
	}
	f := &UserFixture{ctx: ctx, t: t, Users: users, lastIndex: 1}
	f.UserA = f.CreateUser("nick")
	f.UserB = f.CreateUser("dan")
	f.CtxA = auth.ContextWithUser(ctx, f.UserA)
	f.CtxB = auth.ContextWithUser(ctx, f.UserB)
	return f
}

func (f *UserFixture) CreateUser(username string) data.User {
	index := f.lastIndex
	f.lastIndex++

	user, err := f.Users.CreateFromGothUser(f.ctx, data.NewUsername(username), goth.User{
		Provider: "windmill",
		UserID:   fmt.Sprintf("windmill-id-%d", index),
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return user
}
