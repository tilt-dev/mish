// Code generated by protoc-gen-go. DO NOT EDIT.
// source: bridge/fs/proto/fs.proto

package proto

import proto1 "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import wm_data_db "github.com/windmilleng/mish/data/db/proto"
import wm_data_db1 "github.com/windmilleng/mish/data/db/proto"
import wm_data "github.com/windmilleng/mish/data/proto"
import google_protobuf "github.com/golang/protobuf/ptypes/timestamp"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto1.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// snaps create snapshot_dir
type SnapshotDirRequest struct {
	Path           string                  `protobuf:"bytes,1,opt,name=path" json:"path,omitempty"`
	Matcher        *wm_data.Matcher        `protobuf:"bytes,2,opt,name=matcher" json:"matcher,omitempty"`
	BaseSnapshotId string                  `protobuf:"bytes,3,opt,name=base_snapshot_id,json=baseSnapshotId" json:"base_snapshot_id,omitempty"`
	Owner          uint64                  `protobuf:"varint,4,opt,name=owner" json:"owner,omitempty"`
	Tag            *wm_data_db1.RecipeWTag `protobuf:"bytes,5,opt,name=tag" json:"tag,omitempty"`
}

func (m *SnapshotDirRequest) Reset()                    { *m = SnapshotDirRequest{} }
func (m *SnapshotDirRequest) String() string            { return proto1.CompactTextString(m) }
func (*SnapshotDirRequest) ProtoMessage()               {}
func (*SnapshotDirRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{0} }

func (m *SnapshotDirRequest) GetPath() string {
	if m != nil {
		return m.Path
	}
	return ""
}

func (m *SnapshotDirRequest) GetMatcher() *wm_data.Matcher {
	if m != nil {
		return m.Matcher
	}
	return nil
}

func (m *SnapshotDirRequest) GetBaseSnapshotId() string {
	if m != nil {
		return m.BaseSnapshotId
	}
	return ""
}

func (m *SnapshotDirRequest) GetOwner() uint64 {
	if m != nil {
		return m.Owner
	}
	return 0
}

func (m *SnapshotDirRequest) GetTag() *wm_data_db1.RecipeWTag {
	if m != nil {
		return m.Tag
	}
	return nil
}

type SnapshotDirReply struct {
	CreatedSnapId string `protobuf:"bytes,1,opt,name=created_snap_id,json=createdSnapId" json:"created_snap_id,omitempty"`
}

func (m *SnapshotDirReply) Reset()                    { *m = SnapshotDirReply{} }
func (m *SnapshotDirReply) String() string            { return proto1.CompactTextString(m) }
func (*SnapshotDirReply) ProtoMessage()               {}
func (*SnapshotDirReply) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{1} }

func (m *SnapshotDirReply) GetCreatedSnapId() string {
	if m != nil {
		return m.CreatedSnapId
	}
	return ""
}

// snaps checkout
type CheckoutStatus struct {
	Path   string                     `protobuf:"bytes,1,opt,name=path" json:"path,omitempty"`
	SnapId string                     `protobuf:"bytes,2,opt,name=snap_id,json=snapId" json:"snap_id,omitempty"`
	Mtime  *google_protobuf.Timestamp `protobuf:"bytes,3,opt,name=mtime" json:"mtime,omitempty"`
}

func (m *CheckoutStatus) Reset()                    { *m = CheckoutStatus{} }
func (m *CheckoutStatus) String() string            { return proto1.CompactTextString(m) }
func (*CheckoutStatus) ProtoMessage()               {}
func (*CheckoutStatus) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{2} }

func (m *CheckoutStatus) GetPath() string {
	if m != nil {
		return m.Path
	}
	return ""
}

func (m *CheckoutStatus) GetSnapId() string {
	if m != nil {
		return m.SnapId
	}
	return ""
}

func (m *CheckoutStatus) GetMtime() *google_protobuf.Timestamp {
	if m != nil {
		return m.Mtime
	}
	return nil
}

type CheckoutRequest struct {
	SnapId string `protobuf:"bytes,1,opt,name=snap_id,json=snapId" json:"snap_id,omitempty"`
	Path   string `protobuf:"bytes,2,opt,name=path" json:"path,omitempty"`
}

func (m *CheckoutRequest) Reset()                    { *m = CheckoutRequest{} }
func (m *CheckoutRequest) String() string            { return proto1.CompactTextString(m) }
func (*CheckoutRequest) ProtoMessage()               {}
func (*CheckoutRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{3} }

func (m *CheckoutRequest) GetSnapId() string {
	if m != nil {
		return m.SnapId
	}
	return ""
}

func (m *CheckoutRequest) GetPath() string {
	if m != nil {
		return m.Path
	}
	return ""
}

type CheckoutReply struct {
	// DEPRECATED. Use CheckoutStatus.
	CheckoutPath   string          `protobuf:"bytes,1,opt,name=checkout_path,json=checkoutPath" json:"checkout_path,omitempty"`
	CheckoutStatus *CheckoutStatus `protobuf:"bytes,2,opt,name=checkout_status,json=checkoutStatus" json:"checkout_status,omitempty"`
}

func (m *CheckoutReply) Reset()                    { *m = CheckoutReply{} }
func (m *CheckoutReply) String() string            { return proto1.CompactTextString(m) }
func (*CheckoutReply) ProtoMessage()               {}
func (*CheckoutReply) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{4} }

func (m *CheckoutReply) GetCheckoutPath() string {
	if m != nil {
		return m.CheckoutPath
	}
	return ""
}

func (m *CheckoutReply) GetCheckoutStatus() *CheckoutStatus {
	if m != nil {
		return m.CheckoutStatus
	}
	return nil
}

type ResetCheckoutRequest struct {
	// DEPRECATED. Use CheckoutStatus.
	SnapId string          `protobuf:"bytes,1,opt,name=snap_id,json=snapId" json:"snap_id,omitempty"`
	Path   string          `protobuf:"bytes,2,opt,name=path" json:"path,omitempty"`
	Status *CheckoutStatus `protobuf:"bytes,3,opt,name=status" json:"status,omitempty"`
}

func (m *ResetCheckoutRequest) Reset()                    { *m = ResetCheckoutRequest{} }
func (m *ResetCheckoutRequest) String() string            { return proto1.CompactTextString(m) }
func (*ResetCheckoutRequest) ProtoMessage()               {}
func (*ResetCheckoutRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{5} }

func (m *ResetCheckoutRequest) GetSnapId() string {
	if m != nil {
		return m.SnapId
	}
	return ""
}

func (m *ResetCheckoutRequest) GetPath() string {
	if m != nil {
		return m.Path
	}
	return ""
}

func (m *ResetCheckoutRequest) GetStatus() *CheckoutStatus {
	if m != nil {
		return m.Status
	}
	return nil
}

type ResetCheckoutReply struct {
}

func (m *ResetCheckoutReply) Reset()                    { *m = ResetCheckoutReply{} }
func (m *ResetCheckoutReply) String() string            { return proto1.CompactTextString(m) }
func (*ResetCheckoutReply) ProtoMessage()               {}
func (*ResetCheckoutReply) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{6} }

// mirror wm2fs start
type WM2FSStartRequest struct {
	PointerId string `protobuf:"bytes,1,opt,name=pointer_id,json=pointerId" json:"pointer_id,omitempty"`
	Path      string `protobuf:"bytes,2,opt,name=path" json:"path,omitempty"`
}

func (m *WM2FSStartRequest) Reset()                    { *m = WM2FSStartRequest{} }
func (m *WM2FSStartRequest) String() string            { return proto1.CompactTextString(m) }
func (*WM2FSStartRequest) ProtoMessage()               {}
func (*WM2FSStartRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{7} }

func (m *WM2FSStartRequest) GetPointerId() string {
	if m != nil {
		return m.PointerId
	}
	return ""
}

func (m *WM2FSStartRequest) GetPath() string {
	if m != nil {
		return m.Path
	}
	return ""
}

type WM2FSStartReply struct {
}

func (m *WM2FSStartReply) Reset()                    { *m = WM2FSStartReply{} }
func (m *WM2FSStartReply) String() string            { return proto1.CompactTextString(m) }
func (*WM2FSStartReply) ProtoMessage()               {}
func (*WM2FSStartReply) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{8} }

// mirror wm2fs stop
type WM2FSStopRequest struct {
	Path string `protobuf:"bytes,1,opt,name=path" json:"path,omitempty"`
}

func (m *WM2FSStopRequest) Reset()                    { *m = WM2FSStopRequest{} }
func (m *WM2FSStopRequest) String() string            { return proto1.CompactTextString(m) }
func (*WM2FSStopRequest) ProtoMessage()               {}
func (*WM2FSStopRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{9} }

func (m *WM2FSStopRequest) GetPath() string {
	if m != nil {
		return m.Path
	}
	return ""
}

type WM2FSStopReply struct {
}

func (m *WM2FSStopReply) Reset()                    { *m = WM2FSStopReply{} }
func (m *WM2FSStopReply) String() string            { return proto1.CompactTextString(m) }
func (*WM2FSStopReply) ProtoMessage()               {}
func (*WM2FSStopReply) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{10} }

// mirror fs2wm start
type FS2WMStartRequest struct {
	Path      string           `protobuf:"bytes,1,opt,name=path" json:"path,omitempty"`
	PointerId string           `protobuf:"bytes,2,opt,name=pointer_id,json=pointerId" json:"pointer_id,omitempty"`
	Matcher   *wm_data.Matcher `protobuf:"bytes,3,opt,name=matcher" json:"matcher,omitempty"`
}

func (m *FS2WMStartRequest) Reset()                    { *m = FS2WMStartRequest{} }
func (m *FS2WMStartRequest) String() string            { return proto1.CompactTextString(m) }
func (*FS2WMStartRequest) ProtoMessage()               {}
func (*FS2WMStartRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{11} }

func (m *FS2WMStartRequest) GetPath() string {
	if m != nil {
		return m.Path
	}
	return ""
}

func (m *FS2WMStartRequest) GetPointerId() string {
	if m != nil {
		return m.PointerId
	}
	return ""
}

func (m *FS2WMStartRequest) GetMatcher() *wm_data.Matcher {
	if m != nil {
		return m.Matcher
	}
	return nil
}

type FS2WMStartReply struct {
}

func (m *FS2WMStartReply) Reset()                    { *m = FS2WMStartReply{} }
func (m *FS2WMStartReply) String() string            { return proto1.CompactTextString(m) }
func (*FS2WMStartReply) ProtoMessage()               {}
func (*FS2WMStartReply) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{12} }

// mirror fs2wm status
type FS2WMStatusRequest struct {
	// An absolute path to the repo we want to check.
	Path string `protobuf:"bytes,1,opt,name=path" json:"path,omitempty"`
}

func (m *FS2WMStatusRequest) Reset()                    { *m = FS2WMStatusRequest{} }
func (m *FS2WMStatusRequest) String() string            { return proto1.CompactTextString(m) }
func (*FS2WMStatusRequest) ProtoMessage()               {}
func (*FS2WMStatusRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{13} }

func (m *FS2WMStatusRequest) GetPath() string {
	if m != nil {
		return m.Path
	}
	return ""
}

type FS2WMStatusReply struct {
	Status *FsToWmState `protobuf:"bytes,1,opt,name=status" json:"status,omitempty"`
}

func (m *FS2WMStatusReply) Reset()                    { *m = FS2WMStatusReply{} }
func (m *FS2WMStatusReply) String() string            { return proto1.CompactTextString(m) }
func (*FS2WMStatusReply) ProtoMessage()               {}
func (*FS2WMStatusReply) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{14} }

func (m *FS2WMStatusReply) GetStatus() *FsToWmState {
	if m != nil {
		return m.Status
	}
	return nil
}

type FS2WMPointerStatusRequest struct {
	PointerId string `protobuf:"bytes,1,opt,name=pointer_id,json=pointerId" json:"pointer_id,omitempty"`
}

func (m *FS2WMPointerStatusRequest) Reset()                    { *m = FS2WMPointerStatusRequest{} }
func (m *FS2WMPointerStatusRequest) String() string            { return proto1.CompactTextString(m) }
func (*FS2WMPointerStatusRequest) ProtoMessage()               {}
func (*FS2WMPointerStatusRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{15} }

func (m *FS2WMPointerStatusRequest) GetPointerId() string {
	if m != nil {
		return m.PointerId
	}
	return ""
}

type FS2WMPointerStatusReply struct {
	Status *FsToWmState `protobuf:"bytes,1,opt,name=status" json:"status,omitempty"`
}

func (m *FS2WMPointerStatusReply) Reset()                    { *m = FS2WMPointerStatusReply{} }
func (m *FS2WMPointerStatusReply) String() string            { return proto1.CompactTextString(m) }
func (*FS2WMPointerStatusReply) ProtoMessage()               {}
func (*FS2WMPointerStatusReply) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{16} }

func (m *FS2WMPointerStatusReply) GetStatus() *FsToWmState {
	if m != nil {
		return m.Status
	}
	return nil
}

// mirror fs2wm fsync
type FS2WMFSyncRequest struct {
	Pointer string `protobuf:"bytes,1,opt,name=pointer" json:"pointer,omitempty"`
}

func (m *FS2WMFSyncRequest) Reset()                    { *m = FS2WMFSyncRequest{} }
func (m *FS2WMFSyncRequest) String() string            { return proto1.CompactTextString(m) }
func (*FS2WMFSyncRequest) ProtoMessage()               {}
func (*FS2WMFSyncRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{17} }

func (m *FS2WMFSyncRequest) GetPointer() string {
	if m != nil {
		return m.Pointer
	}
	return ""
}

type FS2WMFSyncReply struct {
	Head *wm_data_db.PointerAtSnapshot `protobuf:"bytes,1,opt,name=head" json:"head,omitempty"`
}

func (m *FS2WMFSyncReply) Reset()                    { *m = FS2WMFSyncReply{} }
func (m *FS2WMFSyncReply) String() string            { return proto1.CompactTextString(m) }
func (*FS2WMFSyncReply) ProtoMessage()               {}
func (*FS2WMFSyncReply) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{18} }

func (m *FS2WMFSyncReply) GetHead() *wm_data_db.PointerAtSnapshot {
	if m != nil {
		return m.Head
	}
	return nil
}

// mirror fs2wm stop
type FS2WMStopRequest struct {
	PointerId string `protobuf:"bytes,1,opt,name=pointer_id,json=pointerId" json:"pointer_id,omitempty"`
}

func (m *FS2WMStopRequest) Reset()                    { *m = FS2WMStopRequest{} }
func (m *FS2WMStopRequest) String() string            { return proto1.CompactTextString(m) }
func (*FS2WMStopRequest) ProtoMessage()               {}
func (*FS2WMStopRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{19} }

func (m *FS2WMStopRequest) GetPointerId() string {
	if m != nil {
		return m.PointerId
	}
	return ""
}

type FS2WMStopReply struct {
}

func (m *FS2WMStopReply) Reset()                    { *m = FS2WMStopReply{} }
func (m *FS2WMStopReply) String() string            { return proto1.CompactTextString(m) }
func (*FS2WMStopReply) ProtoMessage()               {}
func (*FS2WMStopReply) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{20} }

func init() {
	proto1.RegisterType((*SnapshotDirRequest)(nil), "wm.fs.SnapshotDirRequest")
	proto1.RegisterType((*SnapshotDirReply)(nil), "wm.fs.SnapshotDirReply")
	proto1.RegisterType((*CheckoutStatus)(nil), "wm.fs.CheckoutStatus")
	proto1.RegisterType((*CheckoutRequest)(nil), "wm.fs.CheckoutRequest")
	proto1.RegisterType((*CheckoutReply)(nil), "wm.fs.CheckoutReply")
	proto1.RegisterType((*ResetCheckoutRequest)(nil), "wm.fs.ResetCheckoutRequest")
	proto1.RegisterType((*ResetCheckoutReply)(nil), "wm.fs.ResetCheckoutReply")
	proto1.RegisterType((*WM2FSStartRequest)(nil), "wm.fs.WM2FSStartRequest")
	proto1.RegisterType((*WM2FSStartReply)(nil), "wm.fs.WM2FSStartReply")
	proto1.RegisterType((*WM2FSStopRequest)(nil), "wm.fs.WM2FSStopRequest")
	proto1.RegisterType((*WM2FSStopReply)(nil), "wm.fs.WM2FSStopReply")
	proto1.RegisterType((*FS2WMStartRequest)(nil), "wm.fs.FS2WMStartRequest")
	proto1.RegisterType((*FS2WMStartReply)(nil), "wm.fs.FS2WMStartReply")
	proto1.RegisterType((*FS2WMStatusRequest)(nil), "wm.fs.FS2WMStatusRequest")
	proto1.RegisterType((*FS2WMStatusReply)(nil), "wm.fs.FS2WMStatusReply")
	proto1.RegisterType((*FS2WMPointerStatusRequest)(nil), "wm.fs.FS2WMPointerStatusRequest")
	proto1.RegisterType((*FS2WMPointerStatusReply)(nil), "wm.fs.FS2WMPointerStatusReply")
	proto1.RegisterType((*FS2WMFSyncRequest)(nil), "wm.fs.FS2WMFSyncRequest")
	proto1.RegisterType((*FS2WMFSyncReply)(nil), "wm.fs.FS2WMFSyncReply")
	proto1.RegisterType((*FS2WMStopRequest)(nil), "wm.fs.FS2WMStopRequest")
	proto1.RegisterType((*FS2WMStopReply)(nil), "wm.fs.FS2WMStopReply")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for WindmillFS service

type WindmillFSClient interface {
	// snaps checkout
	Checkout(ctx context.Context, in *CheckoutRequest, opts ...grpc.CallOption) (*CheckoutReply, error)
	ResetCheckout(ctx context.Context, in *ResetCheckoutRequest, opts ...grpc.CallOption) (*ResetCheckoutReply, error)
	// snaps snapshot_dir
	SnapshotDir(ctx context.Context, in *SnapshotDirRequest, opts ...grpc.CallOption) (*SnapshotDirReply, error)
	// mirror wm2fs start
	// TODO(dbentley): WM2FSS is an awful thing to see/type. Can we define a direction? Like,
	// wm2fs is export, and fs2wm is import?
	WM2FSStart(ctx context.Context, in *WM2FSStartRequest, opts ...grpc.CallOption) (*WM2FSStartReply, error)
	// mirror wm2fs stop
	WM2FSStop(ctx context.Context, in *WM2FSStopRequest, opts ...grpc.CallOption) (*WM2FSStopReply, error)
	// mirror fs2wm start
	//
	// Acquires a pointer, snapshots an existing directory, and sets the
	// pointer to the directory snapshot. Henceforth, any new changes to the
	// directory will create a new snapshot and set the pointer to that snapshot.
	//
	// If the pointer does not exist, creates it.
	FS2WMStart(ctx context.Context, in *FS2WMStartRequest, opts ...grpc.CallOption) (*FS2WMStartReply, error)
	// mirror fs2wm status
	//
	// Prints the status of the mirror at the given path.
	FS2WMStatus(ctx context.Context, in *FS2WMStatusRequest, opts ...grpc.CallOption) (*FS2WMStatusReply, error)
	FS2WMPointerStatus(ctx context.Context, in *FS2WMPointerStatusRequest, opts ...grpc.CallOption) (*FS2WMPointerStatusReply, error)
	// mirror fs2wm fsync
	//
	// Wait for the mirror to catch up with the latest changes to disk.
	FS2WMFSync(ctx context.Context, in *FS2WMFSyncRequest, opts ...grpc.CallOption) (*FS2WMFSyncReply, error)
	// mirror fs2wm stop
	//
	// Stop creating new snapshots when the directory changes.
	FS2WMStop(ctx context.Context, in *FS2WMStopRequest, opts ...grpc.CallOption) (*FS2WMStopReply, error)
}

type windmillFSClient struct {
	cc *grpc.ClientConn
}

func NewWindmillFSClient(cc *grpc.ClientConn) WindmillFSClient {
	return &windmillFSClient{cc}
}

func (c *windmillFSClient) Checkout(ctx context.Context, in *CheckoutRequest, opts ...grpc.CallOption) (*CheckoutReply, error) {
	out := new(CheckoutReply)
	err := grpc.Invoke(ctx, "/wm.fs.WindmillFS/Checkout", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *windmillFSClient) ResetCheckout(ctx context.Context, in *ResetCheckoutRequest, opts ...grpc.CallOption) (*ResetCheckoutReply, error) {
	out := new(ResetCheckoutReply)
	err := grpc.Invoke(ctx, "/wm.fs.WindmillFS/ResetCheckout", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *windmillFSClient) SnapshotDir(ctx context.Context, in *SnapshotDirRequest, opts ...grpc.CallOption) (*SnapshotDirReply, error) {
	out := new(SnapshotDirReply)
	err := grpc.Invoke(ctx, "/wm.fs.WindmillFS/SnapshotDir", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *windmillFSClient) WM2FSStart(ctx context.Context, in *WM2FSStartRequest, opts ...grpc.CallOption) (*WM2FSStartReply, error) {
	out := new(WM2FSStartReply)
	err := grpc.Invoke(ctx, "/wm.fs.WindmillFS/WM2FSStart", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *windmillFSClient) WM2FSStop(ctx context.Context, in *WM2FSStopRequest, opts ...grpc.CallOption) (*WM2FSStopReply, error) {
	out := new(WM2FSStopReply)
	err := grpc.Invoke(ctx, "/wm.fs.WindmillFS/WM2FSStop", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *windmillFSClient) FS2WMStart(ctx context.Context, in *FS2WMStartRequest, opts ...grpc.CallOption) (*FS2WMStartReply, error) {
	out := new(FS2WMStartReply)
	err := grpc.Invoke(ctx, "/wm.fs.WindmillFS/FS2WMStart", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *windmillFSClient) FS2WMStatus(ctx context.Context, in *FS2WMStatusRequest, opts ...grpc.CallOption) (*FS2WMStatusReply, error) {
	out := new(FS2WMStatusReply)
	err := grpc.Invoke(ctx, "/wm.fs.WindmillFS/FS2WMStatus", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *windmillFSClient) FS2WMPointerStatus(ctx context.Context, in *FS2WMPointerStatusRequest, opts ...grpc.CallOption) (*FS2WMPointerStatusReply, error) {
	out := new(FS2WMPointerStatusReply)
	err := grpc.Invoke(ctx, "/wm.fs.WindmillFS/FS2WMPointerStatus", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *windmillFSClient) FS2WMFSync(ctx context.Context, in *FS2WMFSyncRequest, opts ...grpc.CallOption) (*FS2WMFSyncReply, error) {
	out := new(FS2WMFSyncReply)
	err := grpc.Invoke(ctx, "/wm.fs.WindmillFS/FS2WMFSync", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *windmillFSClient) FS2WMStop(ctx context.Context, in *FS2WMStopRequest, opts ...grpc.CallOption) (*FS2WMStopReply, error) {
	out := new(FS2WMStopReply)
	err := grpc.Invoke(ctx, "/wm.fs.WindmillFS/FS2WMStop", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for WindmillFS service

type WindmillFSServer interface {
	// snaps checkout
	Checkout(context.Context, *CheckoutRequest) (*CheckoutReply, error)
	ResetCheckout(context.Context, *ResetCheckoutRequest) (*ResetCheckoutReply, error)
	// snaps snapshot_dir
	SnapshotDir(context.Context, *SnapshotDirRequest) (*SnapshotDirReply, error)
	// mirror wm2fs start
	// TODO(dbentley): WM2FSS is an awful thing to see/type. Can we define a direction? Like,
	// wm2fs is export, and fs2wm is import?
	WM2FSStart(context.Context, *WM2FSStartRequest) (*WM2FSStartReply, error)
	// mirror wm2fs stop
	WM2FSStop(context.Context, *WM2FSStopRequest) (*WM2FSStopReply, error)
	// mirror fs2wm start
	//
	// Acquires a pointer, snapshots an existing directory, and sets the
	// pointer to the directory snapshot. Henceforth, any new changes to the
	// directory will create a new snapshot and set the pointer to that snapshot.
	//
	// If the pointer does not exist, creates it.
	FS2WMStart(context.Context, *FS2WMStartRequest) (*FS2WMStartReply, error)
	// mirror fs2wm status
	//
	// Prints the status of the mirror at the given path.
	FS2WMStatus(context.Context, *FS2WMStatusRequest) (*FS2WMStatusReply, error)
	FS2WMPointerStatus(context.Context, *FS2WMPointerStatusRequest) (*FS2WMPointerStatusReply, error)
	// mirror fs2wm fsync
	//
	// Wait for the mirror to catch up with the latest changes to disk.
	FS2WMFSync(context.Context, *FS2WMFSyncRequest) (*FS2WMFSyncReply, error)
	// mirror fs2wm stop
	//
	// Stop creating new snapshots when the directory changes.
	FS2WMStop(context.Context, *FS2WMStopRequest) (*FS2WMStopReply, error)
}

func RegisterWindmillFSServer(s *grpc.Server, srv WindmillFSServer) {
	s.RegisterService(&_WindmillFS_serviceDesc, srv)
}

func _WindmillFS_Checkout_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CheckoutRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WindmillFSServer).Checkout(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/wm.fs.WindmillFS/Checkout",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WindmillFSServer).Checkout(ctx, req.(*CheckoutRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WindmillFS_ResetCheckout_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ResetCheckoutRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WindmillFSServer).ResetCheckout(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/wm.fs.WindmillFS/ResetCheckout",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WindmillFSServer).ResetCheckout(ctx, req.(*ResetCheckoutRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WindmillFS_SnapshotDir_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SnapshotDirRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WindmillFSServer).SnapshotDir(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/wm.fs.WindmillFS/SnapshotDir",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WindmillFSServer).SnapshotDir(ctx, req.(*SnapshotDirRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WindmillFS_WM2FSStart_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WM2FSStartRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WindmillFSServer).WM2FSStart(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/wm.fs.WindmillFS/WM2FSStart",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WindmillFSServer).WM2FSStart(ctx, req.(*WM2FSStartRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WindmillFS_WM2FSStop_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WM2FSStopRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WindmillFSServer).WM2FSStop(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/wm.fs.WindmillFS/WM2FSStop",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WindmillFSServer).WM2FSStop(ctx, req.(*WM2FSStopRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WindmillFS_FS2WMStart_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FS2WMStartRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WindmillFSServer).FS2WMStart(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/wm.fs.WindmillFS/FS2WMStart",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WindmillFSServer).FS2WMStart(ctx, req.(*FS2WMStartRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WindmillFS_FS2WMStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FS2WMStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WindmillFSServer).FS2WMStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/wm.fs.WindmillFS/FS2WMStatus",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WindmillFSServer).FS2WMStatus(ctx, req.(*FS2WMStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WindmillFS_FS2WMPointerStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FS2WMPointerStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WindmillFSServer).FS2WMPointerStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/wm.fs.WindmillFS/FS2WMPointerStatus",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WindmillFSServer).FS2WMPointerStatus(ctx, req.(*FS2WMPointerStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WindmillFS_FS2WMFSync_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FS2WMFSyncRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WindmillFSServer).FS2WMFSync(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/wm.fs.WindmillFS/FS2WMFSync",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WindmillFSServer).FS2WMFSync(ctx, req.(*FS2WMFSyncRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WindmillFS_FS2WMStop_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FS2WMStopRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WindmillFSServer).FS2WMStop(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/wm.fs.WindmillFS/FS2WMStop",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WindmillFSServer).FS2WMStop(ctx, req.(*FS2WMStopRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _WindmillFS_serviceDesc = grpc.ServiceDesc{
	ServiceName: "wm.fs.WindmillFS",
	HandlerType: (*WindmillFSServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Checkout",
			Handler:    _WindmillFS_Checkout_Handler,
		},
		{
			MethodName: "ResetCheckout",
			Handler:    _WindmillFS_ResetCheckout_Handler,
		},
		{
			MethodName: "SnapshotDir",
			Handler:    _WindmillFS_SnapshotDir_Handler,
		},
		{
			MethodName: "WM2FSStart",
			Handler:    _WindmillFS_WM2FSStart_Handler,
		},
		{
			MethodName: "WM2FSStop",
			Handler:    _WindmillFS_WM2FSStop_Handler,
		},
		{
			MethodName: "FS2WMStart",
			Handler:    _WindmillFS_FS2WMStart_Handler,
		},
		{
			MethodName: "FS2WMStatus",
			Handler:    _WindmillFS_FS2WMStatus_Handler,
		},
		{
			MethodName: "FS2WMPointerStatus",
			Handler:    _WindmillFS_FS2WMPointerStatus_Handler,
		},
		{
			MethodName: "FS2WMFSync",
			Handler:    _WindmillFS_FS2WMFSync_Handler,
		},
		{
			MethodName: "FS2WMStop",
			Handler:    _WindmillFS_FS2WMStop_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "bridge/fs/proto/fs.proto",
}

func init() { proto1.RegisterFile("bridge/fs/proto/fs.proto", fileDescriptor1) }

var fileDescriptor1 = []byte{
	// 839 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xa4, 0x56, 0x5b, 0x4f, 0xf3, 0x46,
	0x10, 0x25, 0x37, 0x28, 0x43, 0x73, 0x61, 0x95, 0x8f, 0x18, 0x53, 0xaa, 0xc8, 0x95, 0x50, 0x84,
	0x84, 0x5d, 0xc2, 0x1b, 0x55, 0x51, 0x0b, 0x34, 0x12, 0x0f, 0x48, 0xc8, 0x41, 0x4a, 0xd5, 0x17,
	0xe4, 0xd8, 0x1b, 0xc7, 0x6a, 0x7c, 0xa9, 0x77, 0xd3, 0x28, 0xbf, 0xad, 0x52, 0x7f, 0x5b, 0xb5,
	0xeb, 0xf5, 0x65, 0x1d, 0x37, 0xa8, 0xea, 0x13, 0xf6, 0xcc, 0xd9, 0x39, 0x67, 0x66, 0xe7, 0x98,
	0x80, 0x32, 0x8f, 0x3d, 0xc7, 0xc5, 0xc6, 0x82, 0x18, 0x51, 0x1c, 0xd2, 0xd0, 0x58, 0x10, 0x9d,
	0x3f, 0xa0, 0xd6, 0xc6, 0xd7, 0x17, 0x44, 0xbd, 0x28, 0x03, 0x08, 0xb5, 0x28, 0x4e, 0x30, 0xea,
	0x37, 0x8e, 0x45, 0x2d, 0xc3, 0x99, 0x8b, 0x54, 0x14, 0x7a, 0x01, 0xc5, 0xb1, 0xa8, 0xa0, 0x5e,
	0xc8, 0xd9, 0x18, 0xdb, 0x5e, 0x84, 0xd3, 0xa4, 0xc2, 0x93, 0x49, 0xc6, 0xb7, 0xa8, 0xbd, 0xc4,
	0xb1, 0xc8, 0x3c, 0xfe, 0x89, 0x03, 0x27, 0x8c, 0x0d, 0xd7, 0xa3, 0xcb, 0xf5, 0x5c, 0xb7, 0x43,
	0xdf, 0x70, 0xc3, 0x95, 0x15, 0xb8, 0x09, 0x7a, 0xbe, 0x5e, 0x18, 0x11, 0xdd, 0x46, 0x98, 0x18,
	0xd4, 0xf3, 0x31, 0xa1, 0x96, 0x1f, 0xe5, 0x4f, 0x49, 0x0d, 0xed, 0xef, 0x1a, 0xa0, 0x69, 0x60,
	0x45, 0x64, 0x19, 0xd2, 0x67, 0x2f, 0x36, 0xf1, 0x1f, 0x6b, 0x4c, 0x28, 0x42, 0xd0, 0x8c, 0x2c,
	0xba, 0x54, 0x6a, 0xc3, 0xda, 0xe8, 0xd8, 0xe4, 0xcf, 0xe8, 0x1a, 0x8e, 0x04, 0xbf, 0x52, 0x1f,
	0xd6, 0x46, 0x27, 0xe3, 0x9e, 0xbe, 0xf1, 0x75, 0xa6, 0x4e, 0x7f, 0x4d, 0xe2, 0x66, 0x0a, 0x40,
	0x23, 0xe8, 0xcd, 0x2d, 0x82, 0x3f, 0x88, 0x28, 0xfd, 0xe1, 0x39, 0x4a, 0x83, 0xd7, 0xea, 0xb0,
	0x78, 0xca, 0xf8, 0xe2, 0xa0, 0x3e, 0xb4, 0xc2, 0x4d, 0x80, 0x63, 0xa5, 0x39, 0xac, 0x8d, 0x9a,
	0x66, 0xf2, 0x82, 0x46, 0xd0, 0xa0, 0x96, 0xab, 0xb4, 0x38, 0xcf, 0x59, 0xc6, 0xe3, 0xcc, 0x75,
	0x93, 0x0f, 0x67, 0xf6, 0x6e, 0xb9, 0x26, 0x83, 0x68, 0xf7, 0xd0, 0x93, 0xf4, 0x47, 0xab, 0x2d,
	0xba, 0x82, 0xae, 0x1d, 0x63, 0x8b, 0x62, 0x87, 0x0b, 0x60, 0xe4, 0x49, 0x23, 0x6d, 0x11, 0x66,
	0x27, 0x5e, 0x1c, 0x2d, 0x84, 0xce, 0xd3, 0x12, 0xdb, 0xbf, 0x87, 0x6b, 0x3a, 0xa5, 0x16, 0x5d,
	0x93, 0xca, 0xbe, 0x07, 0x70, 0x94, 0x56, 0xa9, 0xf3, 0xf0, 0x21, 0xe1, 0xc7, 0xd1, 0xf7, 0xd0,
	0xf2, 0xd9, 0x3c, 0x79, 0x67, 0x27, 0x63, 0x55, 0x77, 0xc3, 0xd0, 0x5d, 0x89, 0x2b, 0x9f, 0xaf,
	0x17, 0xfa, 0x7b, 0x3a, 0x6c, 0x33, 0x01, 0x6a, 0x0f, 0xd0, 0x4d, 0x09, 0xd3, 0x49, 0x17, 0xaa,
	0xd7, 0xa4, 0xea, 0xa9, 0x94, 0x7a, 0x2e, 0x45, 0xa3, 0xd0, 0xce, 0xcf, 0xb3, 0x4e, 0xbf, 0x83,
	0xb6, 0x2d, 0x02, 0x1f, 0x05, 0xe1, 0x5f, 0xa7, 0xc1, 0x37, 0xd6, 0xc0, 0x03, 0x74, 0x33, 0x10,
	0xe1, 0x7d, 0x8a, 0x0b, 0xfc, 0xa2, 0xf3, 0xd5, 0xd5, 0xe5, 0x21, 0x98, 0x1d, 0x5b, 0x7a, 0xd7,
	0x62, 0xe8, 0x9b, 0x98, 0x60, 0xfa, 0x7f, 0xa4, 0xa3, 0x1b, 0x38, 0x14, 0xdc, 0x8d, 0x7d, 0xdc,
	0x02, 0xa4, 0xf5, 0x01, 0x95, 0x38, 0xa3, 0xd5, 0x56, 0x9b, 0xc0, 0xe9, 0xec, 0x75, 0x3c, 0x99,
	0x4e, 0xa9, 0x15, 0x67, 0x32, 0x2e, 0x01, 0x84, 0x9f, 0x72, 0x25, 0xc7, 0x22, 0xf2, 0x2f, 0x73,
	0x3c, 0x85, 0x6e, 0xb1, 0x0e, 0x2b, 0x7d, 0x05, 0x3d, 0x11, 0x0a, 0xa3, 0x3d, 0x2e, 0xd0, 0x7a,
	0xd0, 0x29, 0xe0, 0xd8, 0xc9, 0x18, 0x4e, 0x27, 0xd3, 0xf1, 0xec, 0x55, 0x12, 0x55, 0xb5, 0x48,
	0xb2, 0xd0, 0x7a, 0x59, 0x68, 0xc1, 0x5f, 0x8d, 0x4f, 0xfc, 0xc5, 0x1a, 0x28, 0x72, 0x32, 0x19,
	0x23, 0x40, 0x69, 0x88, 0x0d, 0x72, 0x4f, 0x0b, 0x0f, 0xd0, 0x93, 0x90, 0x6c, 0x91, 0xae, 0xb3,
	0xeb, 0xa9, 0x71, 0x6e, 0x24, 0xae, 0x67, 0x42, 0xde, 0xc3, 0x99, 0xcf, 0x90, 0x38, 0xbb, 0x9b,
	0x7b, 0x38, 0xe7, 0xe7, 0xdf, 0x12, 0xe9, 0x32, 0xe1, 0xfe, 0xdb, 0xd0, 0x7e, 0x81, 0x41, 0xd5,
	0xd9, 0xff, 0x2a, 0xe1, 0x46, 0xcc, 0x7c, 0x32, 0xdd, 0x06, 0x76, 0x4a, 0xad, 0xc0, 0x91, 0x20,
	0x12, 0xbc, 0xe9, 0xab, 0xf6, 0x2c, 0xc6, 0x25, 0xe0, 0x8c, 0xed, 0x16, 0x9a, 0x4b, 0x6c, 0x39,
	0x82, 0xeb, 0xb2, 0xf8, 0x89, 0x11, 0xda, 0x7e, 0xa6, 0xe9, 0x87, 0xc5, 0xe4, 0x50, 0xed, 0x36,
	0x9b, 0x5b, 0xbe, 0x22, 0x9f, 0xb4, 0xdb, 0x83, 0x4e, 0xe1, 0x48, 0xb4, 0xda, 0x8e, 0xff, 0x6a,
	0x01, 0xcc, 0xbc, 0xc0, 0xf1, 0xbd, 0xd5, 0x6a, 0x32, 0x45, 0xf7, 0xf0, 0x55, 0xba, 0xe2, 0xe8,
	0xac, 0x64, 0x09, 0xc1, 0xa1, 0xf6, 0x77, 0xe2, 0xec, 0xbe, 0x0f, 0xd0, 0x0b, 0xb4, 0x25, 0x8f,
	0xa0, 0x0b, 0x01, 0xac, 0x72, 0xab, 0x7a, 0x5e, 0x9d, 0x4c, 0x4a, 0x3d, 0xc1, 0x49, 0xe1, 0x2b,
	0x8a, 0x52, 0xec, 0xee, 0x7f, 0x06, 0x75, 0x50, 0x95, 0x4a, 0x8a, 0xfc, 0x04, 0x90, 0xbb, 0x0a,
	0x29, 0x02, 0xb8, 0x63, 0x58, 0xf5, 0xac, 0x22, 0x93, 0x54, 0xf8, 0x11, 0x8e, 0x33, 0x73, 0xa1,
	0x81, 0x0c, 0xcb, 0x66, 0xae, 0x7e, 0xd9, 0x4d, 0x64, 0x02, 0x72, 0x57, 0x64, 0x02, 0x76, 0xcc,
	0x99, 0x09, 0x28, 0x5b, 0x88, 0xcf, 0xa1, 0x60, 0x8d, 0x6c, 0x0e, 0xbb, 0xc6, 0xca, 0xe6, 0x50,
	0x76, 0x92, 0x76, 0x80, 0x7e, 0x15, 0x4e, 0x94, 0x76, 0x1c, 0x0d, 0x8b, 0x07, 0xaa, 0xac, 0xa3,
	0x7e, 0xbb, 0x07, 0x21, 0x37, 0xc8, 0xf7, 0x58, 0x6e, 0xb0, 0xe8, 0x04, 0xb9, 0xc1, 0x7c, 0xe9,
	0x93, 0x09, 0x67, 0x0b, 0x89, 0x4a, 0x3d, 0xec, 0x4e, 0x58, 0xde, 0x5d, 0xed, 0xe0, 0xf1, 0xee,
	0xb7, 0xdb, 0xc2, 0xaf, 0x8d, 0x8d, 0xd8, 0x63, 0x1c, 0xb8, 0x86, 0xef, 0x91, 0xa5, 0x51, 0xfa,
	0x09, 0xf4, 0x43, 0xf2, 0x9f, 0xf0, 0x90, 0xff, 0xb9, 0xfb, 0x27, 0x00, 0x00, 0xff, 0xff, 0xb2,
	0xea, 0x6d, 0x2f, 0x43, 0x09, 0x00, 0x00,
}
