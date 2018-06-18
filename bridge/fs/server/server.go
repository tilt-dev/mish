package server

import (
	"fmt"
	"path/filepath"

	oldctx "golang.org/x/net/context"

	"github.com/windmilleng/mish/bridge/fs"
	"github.com/windmilleng/mish/bridge/fs/proto"
	"github.com/windmilleng/mish/data"
	dbProto "github.com/windmilleng/mish/data/db/proto"
	ospathConv "github.com/windmilleng/mish/os/ospath/convert"
)

type FSServer struct {
	f     fs.FSBridge
	owner data.UserID
}

func NewFSServer(f fs.FSBridge) *FSServer {
	return &FSServer{f: f}
}

// snaps create snapshot_dir
func (s *FSServer) SnapshotDir(ctx oldctx.Context, req *proto.SnapshotDirRequest) (*proto.SnapshotDirReply, error) {
	if !filepath.IsAbs(req.Path) {
		return nil, fmt.Errorf("Path is not absolute: %s", req.Path)
	}

	matcher, err := ospathConv.MatcherP2D(req.GetMatcher())
	if err != nil {
		return nil, err
	}

	created, err := s.f.SnapshotDir(ctx, req.Path, matcher, data.UserID(req.Owner),
		dbProto.RecipeWTagP2D(req.Tag), fs.Hint{Base: data.ParseSnapshotID(req.BaseSnapshotId)})
	if err != nil {
		return nil, err
	}

	return &proto.SnapshotDirReply{CreatedSnapId: created.String()}, nil
}

// snaps checkout
func (s *FSServer) Checkout(ctx oldctx.Context, req *proto.CheckoutRequest) (*proto.CheckoutReply, error) {
	status, err := s.f.Checkout(ctx, data.ParseSnapshotID(req.SnapId), req.Path)
	if err != nil {
		return nil, err
	}

	statusProto, err := fs.CheckoutStatusD2P(status)
	if err != nil {
		return nil, err
	}

	return &proto.CheckoutReply{
		CheckoutPath:   status.Path,
		CheckoutStatus: statusProto,
	}, nil
}

func (s *FSServer) ResetCheckout(ctx oldctx.Context, req *proto.ResetCheckoutRequest) (*proto.ResetCheckoutReply, error) {
	status := fs.CheckoutStatus{
		Path:   req.Path,
		SnapID: data.ParseSnapshotID(req.SnapId),
	}

	if req.Status != nil {
		var err error
		status, err = fs.CheckoutStatusP2D(req.Status)
		if err != nil {
			return nil, err
		}
	}

	err := s.f.ResetCheckout(ctx, status)
	if err != nil {
		return nil, err
	}
	return &proto.ResetCheckoutReply{}, nil
}

// mirror

// mirror wm2fs

// mirror wm2fs start
func (s *FSServer) WM2FSStart(ctx oldctx.Context, req *proto.WM2FSStartRequest) (*proto.WM2FSStartReply, error) {
	if !filepath.IsAbs(req.Path) {
		return nil, fmt.Errorf("Path is not absolute: %s", req.Path)
	}

	id, err := data.ParsePointerID(req.PointerId)
	if err != nil {
		return nil, err
	}

	err = s.f.FromWMStart(ctx, id, req.Path)
	if err != nil {
		return nil, err
	}

	return &proto.WM2FSStartReply{}, nil
}

// mirror wm2fs stop
func (s *FSServer) WM2FSStop(ctx oldctx.Context, req *proto.WM2FSStopRequest) (*proto.WM2FSStopReply, error) {
	if !filepath.IsAbs(req.Path) {
		return nil, fmt.Errorf("Path is not absolute: %s", req.Path)
	}

	err := s.f.FromWMStop(ctx, req.Path)
	if err != nil {
		return nil, err
	}

	return &proto.WM2FSStopReply{}, nil
}

// mirror fs2wm

// mirror fs2wm start
func (s *FSServer) FS2WMStart(ctx oldctx.Context, req *proto.FS2WMStartRequest) (*proto.FS2WMStartReply, error) {
	if !filepath.IsAbs(req.Path) {
		return nil, fmt.Errorf("Path is not absolute: %s", req.Path)
	}

	matcher, err := ospathConv.MatcherP2D(req.Matcher)
	if err != nil {
		return nil, err
	}

	if len(matcher.ToPatterns()) == 0 {
		return nil, fmt.Errorf("Ignore patterns should not be empty")
	}

	id, err := data.ParsePointerID(req.PointerId)
	if err != nil {
		return nil, err
	}

	err = s.f.ToWMStart(ctx, req.Path, id, matcher)
	if err != nil {
		return nil, err
	}

	return &proto.FS2WMStartReply{}, nil
}

// mirror fs2wm fsync
func (s *FSServer) FS2WMFSync(ctx oldctx.Context, req *proto.FS2WMFSyncRequest) (*proto.FS2WMFSyncReply, error) {
	id, err := data.ParsePointerID(req.Pointer)
	if err != nil {
		return nil, err
	}

	head, err := s.f.ToWMFSync(ctx, id)
	if err != nil {
		return nil, err
	}

	return &proto.FS2WMFSyncReply{
		Head: dbProto.PointerAtSnapshotD2P(head),
	}, nil
}

// mirror fs2wm status
func (s *FSServer) FS2WMStatus(ctx oldctx.Context, req *proto.FS2WMStatusRequest) (*proto.FS2WMStatusReply, error) {
	if !filepath.IsAbs(req.Path) {
		return nil, fmt.Errorf("Path is not absolute: %s", req.Path)
	}

	status, err := s.f.ToWMStatus(ctx, req.Path)
	if err != nil {
		return nil, err
	}

	return &proto.FS2WMStatusReply{Status: status}, nil
}

func (s *FSServer) FS2WMPointerStatus(ctx oldctx.Context, req *proto.FS2WMPointerStatusRequest) (*proto.FS2WMPointerStatusReply, error) {
	id, err := data.ParsePointerID(req.PointerId)
	if err != nil {
		return nil, err
	}

	status, err := s.f.ToWMPointerStatus(ctx, id)
	if err != nil {
		return nil, err
	}

	return &proto.FS2WMPointerStatusReply{Status: status}, nil
}

// mirror fs2wm stop
func (s *FSServer) FS2WMStop(ctx oldctx.Context, req *proto.FS2WMStopRequest) (*proto.FS2WMStopReply, error) {
	id, err := data.ParsePointerID(req.PointerId)
	if err != nil {
		return nil, err
	}
	err = s.f.ToWMStop(ctx, id)
	if err != nil {
		return nil, err
	}

	return &proto.FS2WMStopReply{}, nil
}
