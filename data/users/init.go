package users

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/markbates/goth"
	"github.com/windmilleng/mish/data"
)

var defaultGithubLogins = []string{
	"nicks",
	"dbentley",
	"jazzdan",
	"yuindustries",
	"maiamcc",
	"pmt",
}

// Create native accounts.
func initStore(ctx context.Context, users data.UserStore) error {
	// It's OK if the users have already been created. This is expected when
	// connecting to a shared postgres db.
	_, err := users.CreateFromGothUser(ctx, data.Root, goth.User{Provider: "windmill", UserID: string(data.Root)})
	if err != nil && grpc.Code(err) != codes.AlreadyExists {
		return err
	}

	_, err = users.CreateFromGothUser(ctx, data.Anonymous, goth.User{Provider: "windmill", UserID: string(data.Anonymous)})
	if err != nil && grpc.Code(err) != codes.AlreadyExists {
		return err
	}

	_, err = users.CreateFromGothUser(ctx, data.Public, goth.User{Provider: "windmill", UserID: string(data.Public)})
	if err != nil && grpc.Code(err) != codes.AlreadyExists {
		return err
	}

	for _, login := range defaultGithubLogins {
		err := users.WhitelistGithubLogin(ctx, login)
		if err != nil {
			return err
		}
	}
	return nil
}
