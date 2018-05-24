package users

import (
	"context"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/proto"
)

type GRPCAuth struct {
	cl proto.AuthenticatorClient
}

func NewGRPCAuth(cl proto.AuthenticatorClient) GRPCAuth {
	return GRPCAuth{cl: cl}
}

func (a GRPCAuth) CurrentUser(ctx context.Context) (data.User, error) {
	resp, err := a.cl.CurrentUser(ctx, &proto.CurrentUserRequest{})
	if err != nil {
		return data.User{}, err
	}

	return proto.UserP2D(resp.User), nil
}
