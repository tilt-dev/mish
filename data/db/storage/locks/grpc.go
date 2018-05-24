package locks

import (
	"context"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/data/db/proto"
	"github.com/windmilleng/mish/data/db/storage"
)

type GRPCPointerLocker struct {
	cl proto.PointerLockerClient
}

func NewGRPCPointerLocker(cl proto.PointerLockerClient) GRPCPointerLocker {
	return GRPCPointerLocker{cl: cl}
}

func (l GRPCPointerLocker) IsHoldingPointer(ctx context.Context, id data.PointerID) (bool, error) {
	reply, err := l.cl.IsHoldingPointer(ctx, &proto.IsHoldingPointerRequest{
		Id: id.String(),
	})
	if err != nil {
		return false, err
	}
	return reply.Holding, nil
}

var _ storage.PointerLocker = GRPCPointerLocker{}
