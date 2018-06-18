package fss

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/windmilleng/mish/bridge/fs"
	"github.com/windmilleng/mish/bridge/fs/proto"
	"github.com/windmilleng/mish/data"
	dbProto "github.com/windmilleng/mish/data/db/proto"
	"github.com/windmilleng/mish/os/ospath"
	ospathConv "github.com/windmilleng/mish/os/ospath/convert"
)

type GRPCFSBridge struct {
	cl proto.WindmillFSClient
}

func NewGRPCFSBridge(cl proto.WindmillFSClient) *GRPCFSBridge {
	return &GRPCFSBridge{cl: cl}
}

func (b *GRPCFSBridge) Checkout(ctx context.Context, id data.SnapshotID, path string) (fs.CheckoutStatus, error) {
	req := &proto.CheckoutRequest{
		SnapId: id.String(),
		Path:   path,
	}

	resp, err := b.cl.Checkout(ctx, req)
	if err != nil {
		return fs.CheckoutStatus{}, err
	}

	status := fs.CheckoutStatus{
		Path:   resp.CheckoutPath,
		SnapID: id,
	}

	if resp.CheckoutStatus != nil {
		var err error
		status, err = fs.CheckoutStatusP2D(resp.CheckoutStatus)
		if err != nil {
			return fs.CheckoutStatus{}, err
		}
	}

	return status, nil
}

func (b *GRPCFSBridge) ResetCheckout(ctx context.Context, status fs.CheckoutStatus) error {
	statusProto, err := fs.CheckoutStatusD2P(status)
	if err != nil {
		return err
	}

	req := &proto.ResetCheckoutRequest{
		SnapId: status.SnapID.String(),
		Path:   status.Path,
		Status: statusProto,
	}

	_, err = b.cl.ResetCheckout(ctx, req)
	return err
}

func (b *GRPCFSBridge) SnapshotDir(ctx context.Context, path string, matcher *ospath.Matcher, owner data.UserID, tag data.RecipeWTag, hint fs.Hint) (data.SnapshotID, error) {
	resp, err := b.cl.SnapshotDir(ctx, &proto.SnapshotDirRequest{
		Path:           path,
		Matcher:        ospathConv.MatcherD2P(matcher),
		BaseSnapshotId: hint.Base.String(),
		Owner:          uint64(owner),
		Tag:            dbProto.RecipeWTagD2P(tag),
	})
	if err != nil {
		return data.SnapshotID{}, err
	}

	return data.ParseSnapshotID(resp.CreatedSnapId), nil
}

func (b *GRPCFSBridge) ToWMStart(ctx context.Context, path string, ptr data.PointerID, matcher *ospath.Matcher) error {
	req := &proto.FS2WMStartRequest{
		Path:      path,
		PointerId: ptr.String(),
		Matcher:   ospathConv.MatcherD2P(matcher),
	}

	_, err := b.cl.FS2WMStart(ctx, req)
	return err
}

func (b *GRPCFSBridge) ToWMStatus(ctx context.Context, path string) (*proto.FsToWmState, error) {
	req := &proto.FS2WMStatusRequest{
		Path: path,
	}

	reply, err := b.cl.FS2WMStatus(ctx, req)
	if err != nil {
		return nil, err
	}
	return reply.Status, nil
}

func (b *GRPCFSBridge) ToWMPointerStatus(ctx context.Context, id data.PointerID) (*proto.FsToWmState, error) {
	req := &proto.FS2WMPointerStatusRequest{
		PointerId: id.String(),
	}

	reply, err := b.cl.FS2WMPointerStatus(ctx, req)
	if err != nil {
		return nil, err
	}
	return reply.Status, nil
}

func (b *GRPCFSBridge) ToWMFSync(ctx context.Context, ptr data.PointerID) (data.PointerAtSnapshot, error) {
	req := &proto.FS2WMFSyncRequest{
		Pointer: ptr.String(),
	}

	reply, err := b.cl.FS2WMFSync(ctx, req)
	if err != nil {
		return data.PointerAtSnapshot{}, err
	}

	return dbProto.PointerAtSnapshotP2D(reply.Head)
}

func (b *GRPCFSBridge) ToWMStop(ctx context.Context, ptr data.PointerID) error {
	req := &proto.FS2WMStopRequest{
		PointerId: ptr.String(),
	}

	_, err := b.cl.FS2WMStop(ctx, req)
	return err
}

func (b *GRPCFSBridge) FromWMStart(ctx context.Context, ptr data.PointerID, path string) error {
	req := &proto.WM2FSStartRequest{
		PointerId: ptr.String(),
		Path:      path,
	}

	_, err := b.cl.WM2FSStart(ctx, req)
	return err
}

func (b *GRPCFSBridge) FromWMStop(ctx context.Context, path string) error {
	req := &proto.WM2FSStopRequest{
		Path: path,
	}

	_, err := b.cl.WM2FSStop(ctx, req)
	return err
}

func (b *GRPCFSBridge) Shutdown(ctx context.Context) error {
	return grpc.Errorf(codes.PermissionDenied, "Remote shutdown not allowed")
}

var _ fs.FSBridge = &GRPCFSBridge{}
