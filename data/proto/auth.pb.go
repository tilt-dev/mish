// Code generated by protoc-gen-go. DO NOT EDIT.
// source: data/proto/auth.proto

/*
Package proto is a generated protocol buffer package.

It is generated from these files:
	data/proto/auth.proto
	data/proto/matcher.proto
	data/proto/user.proto

It has these top-level messages:
	CurrentUserRequest
	CurrentUserReply
	Matcher
	User
*/
package proto

import proto1 "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto1.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto1.ProtoPackageIsVersion2 // please upgrade the proto package

type CurrentUserRequest struct {
}

func (m *CurrentUserRequest) Reset()                    { *m = CurrentUserRequest{} }
func (m *CurrentUserRequest) String() string            { return proto1.CompactTextString(m) }
func (*CurrentUserRequest) ProtoMessage()               {}
func (*CurrentUserRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type CurrentUserReply struct {
	User *User `protobuf:"bytes,1,opt,name=user" json:"user,omitempty"`
}

func (m *CurrentUserReply) Reset()                    { *m = CurrentUserReply{} }
func (m *CurrentUserReply) String() string            { return proto1.CompactTextString(m) }
func (*CurrentUserReply) ProtoMessage()               {}
func (*CurrentUserReply) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *CurrentUserReply) GetUser() *User {
	if m != nil {
		return m.User
	}
	return nil
}

func init() {
	proto1.RegisterType((*CurrentUserRequest)(nil), "wm.data.CurrentUserRequest")
	proto1.RegisterType((*CurrentUserReply)(nil), "wm.data.CurrentUserReply")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for Authenticator service

type AuthenticatorClient interface {
	CurrentUser(ctx context.Context, in *CurrentUserRequest, opts ...grpc.CallOption) (*CurrentUserReply, error)
}

type authenticatorClient struct {
	cc *grpc.ClientConn
}

func NewAuthenticatorClient(cc *grpc.ClientConn) AuthenticatorClient {
	return &authenticatorClient{cc}
}

func (c *authenticatorClient) CurrentUser(ctx context.Context, in *CurrentUserRequest, opts ...grpc.CallOption) (*CurrentUserReply, error) {
	out := new(CurrentUserReply)
	err := grpc.Invoke(ctx, "/wm.data.Authenticator/CurrentUser", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Authenticator service

type AuthenticatorServer interface {
	CurrentUser(context.Context, *CurrentUserRequest) (*CurrentUserReply, error)
}

func RegisterAuthenticatorServer(s *grpc.Server, srv AuthenticatorServer) {
	s.RegisterService(&_Authenticator_serviceDesc, srv)
}

func _Authenticator_CurrentUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CurrentUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthenticatorServer).CurrentUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/wm.data.Authenticator/CurrentUser",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthenticatorServer).CurrentUser(ctx, req.(*CurrentUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Authenticator_serviceDesc = grpc.ServiceDesc{
	ServiceName: "wm.data.Authenticator",
	HandlerType: (*AuthenticatorServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CurrentUser",
			Handler:    _Authenticator_CurrentUser_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "data/proto/auth.proto",
}

func init() { proto1.RegisterFile("data/proto/auth.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 193 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x4d, 0x49, 0x2c, 0x49,
	0xd4, 0x2f, 0x28, 0xca, 0x2f, 0xc9, 0xd7, 0x4f, 0x2c, 0x2d, 0xc9, 0xd0, 0x03, 0x33, 0x85, 0xd8,
	0xcb, 0x73, 0xf5, 0x40, 0x32, 0x52, 0xc8, 0xf2, 0xa5, 0xc5, 0xa9, 0x45, 0x10, 0x79, 0x25, 0x11,
	0x2e, 0x21, 0xe7, 0xd2, 0xa2, 0xa2, 0xd4, 0xbc, 0x92, 0xd0, 0xe2, 0xd4, 0xa2, 0xa0, 0xd4, 0xc2,
	0xd2, 0xd4, 0xe2, 0x12, 0x25, 0x53, 0x2e, 0x01, 0x14, 0xd1, 0x82, 0x9c, 0x4a, 0x21, 0x45, 0x2e,
	0x16, 0x90, 0x3e, 0x09, 0x46, 0x05, 0x46, 0x0d, 0x6e, 0x23, 0x5e, 0x3d, 0xa8, 0xc1, 0x7a, 0x60,
	0x15, 0x60, 0x29, 0xa3, 0x08, 0x2e, 0x5e, 0xc7, 0xd2, 0x92, 0x8c, 0xd4, 0xbc, 0x92, 0xcc, 0xe4,
	0xc4, 0x92, 0xfc, 0x22, 0x21, 0x77, 0x2e, 0x6e, 0x24, 0x73, 0x84, 0xa4, 0xe1, 0x9a, 0x30, 0xed,
	0x94, 0x92, 0xc4, 0x2e, 0x59, 0x90, 0x53, 0xa9, 0xc4, 0xe0, 0xa4, 0x17, 0xa5, 0x93, 0x9e, 0x59,
	0x92, 0x51, 0x9a, 0xa4, 0x97, 0x9c, 0x9f, 0xab, 0x5f, 0x9e, 0x99, 0x97, 0x92, 0x9b, 0x99, 0x93,
	0x93, 0x9a, 0x97, 0xae, 0x9f, 0x9b, 0x59, 0x9c, 0xa1, 0x8f, 0xf0, 0x9b, 0x35, 0x98, 0x4c, 0x62,
	0x03, 0x53, 0xc6, 0x80, 0x00, 0x00, 0x00, 0xff, 0xff, 0x6d, 0xf4, 0xdf, 0xe2, 0x16, 0x01, 0x00,
	0x00,
}
