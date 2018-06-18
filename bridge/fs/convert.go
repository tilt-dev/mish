package fs

import (
	"github.com/golang/protobuf/ptypes"
	"github.com/windmilleng/mish/bridge/fs/proto"
	"github.com/windmilleng/mish/data"
	ospathConv "github.com/windmilleng/mish/os/ospath/convert"
)

func SnapshotConfigP2D(p *proto.SnapshotConfig) (*SnapshotConfig, error) {
	if p == nil {
		return nil, nil
	}
	matcher, err := ospathConv.MatcherP2D(p.Matcher)
	if err != nil {
		return nil, err
	}
	return NewSnapshotConfig(matcher), nil
}

func SnapshotConfigD2P(c *SnapshotConfig) *proto.SnapshotConfig {
	if c == nil {
		return nil
	}
	return &proto.SnapshotConfig{
		Matcher: ospathConv.MatcherD2P(c.Matcher()),
	}
}

func CheckoutStatusP2D(p *proto.CheckoutStatus) (CheckoutStatus, error) {
	if p == nil {
		return CheckoutStatus{}, nil
	}
	mtime, err := ptypes.Timestamp(p.Mtime)
	if err != nil {
		return CheckoutStatus{}, nil
	}
	return CheckoutStatus{
		Path:   p.Path,
		SnapID: data.ParseSnapshotID(p.SnapId),
		Mtime:  mtime,
	}, nil
}

func CheckoutStatusD2P(s CheckoutStatus) (*proto.CheckoutStatus, error) {
	mtime, err := ptypes.TimestampProto(s.Mtime)
	if err != nil {
		return nil, nil
	}
	return &proto.CheckoutStatus{
		Path:   s.Path,
		SnapId: s.SnapID.String(),
		Mtime:  mtime,
	}, nil
}
