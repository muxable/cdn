// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.5
// source: cdn.proto

package api

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// CDNClient is the client API for CDN service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type CDNClient interface {
	Publish(ctx context.Context, opts ...grpc.CallOption) (CDN_PublishClient, error)
	Subscribe(ctx context.Context, opts ...grpc.CallOption) (CDN_SubscribeClient, error)
}

type cDNClient struct {
	cc grpc.ClientConnInterface
}

func NewCDNClient(cc grpc.ClientConnInterface) CDNClient {
	return &cDNClient{cc}
}

func (c *cDNClient) Publish(ctx context.Context, opts ...grpc.CallOption) (CDN_PublishClient, error) {
	stream, err := c.cc.NewStream(ctx, &CDN_ServiceDesc.Streams[0], "/api.CDN/Publish", opts...)
	if err != nil {
		return nil, err
	}
	x := &cDNPublishClient{stream}
	return x, nil
}

type CDN_PublishClient interface {
	Send(*PublishRequest) error
	Recv() (*PublishResponse, error)
	grpc.ClientStream
}

type cDNPublishClient struct {
	grpc.ClientStream
}

func (x *cDNPublishClient) Send(m *PublishRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *cDNPublishClient) Recv() (*PublishResponse, error) {
	m := new(PublishResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *cDNClient) Subscribe(ctx context.Context, opts ...grpc.CallOption) (CDN_SubscribeClient, error) {
	stream, err := c.cc.NewStream(ctx, &CDN_ServiceDesc.Streams[1], "/api.CDN/Subscribe", opts...)
	if err != nil {
		return nil, err
	}
	x := &cDNSubscribeClient{stream}
	return x, nil
}

type CDN_SubscribeClient interface {
	Send(*SubscribeRequest) error
	Recv() (*SubscribeResponse, error)
	grpc.ClientStream
}

type cDNSubscribeClient struct {
	grpc.ClientStream
}

func (x *cDNSubscribeClient) Send(m *SubscribeRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *cDNSubscribeClient) Recv() (*SubscribeResponse, error) {
	m := new(SubscribeResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// CDNServer is the server API for CDN service.
// All implementations should embed UnimplementedCDNServer
// for forward compatibility
type CDNServer interface {
	Publish(CDN_PublishServer) error
	Subscribe(CDN_SubscribeServer) error
}

// UnimplementedCDNServer should be embedded to have forward compatible implementations.
type UnimplementedCDNServer struct {
}

func (UnimplementedCDNServer) Publish(CDN_PublishServer) error {
	return status.Errorf(codes.Unimplemented, "method Publish not implemented")
}
func (UnimplementedCDNServer) Subscribe(CDN_SubscribeServer) error {
	return status.Errorf(codes.Unimplemented, "method Subscribe not implemented")
}

// UnsafeCDNServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to CDNServer will
// result in compilation errors.
type UnsafeCDNServer interface {
	mustEmbedUnimplementedCDNServer()
}

func RegisterCDNServer(s grpc.ServiceRegistrar, srv CDNServer) {
	s.RegisterService(&CDN_ServiceDesc, srv)
}

func _CDN_Publish_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(CDNServer).Publish(&cDNPublishServer{stream})
}

type CDN_PublishServer interface {
	Send(*PublishResponse) error
	Recv() (*PublishRequest, error)
	grpc.ServerStream
}

type cDNPublishServer struct {
	grpc.ServerStream
}

func (x *cDNPublishServer) Send(m *PublishResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *cDNPublishServer) Recv() (*PublishRequest, error) {
	m := new(PublishRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _CDN_Subscribe_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(CDNServer).Subscribe(&cDNSubscribeServer{stream})
}

type CDN_SubscribeServer interface {
	Send(*SubscribeResponse) error
	Recv() (*SubscribeRequest, error)
	grpc.ServerStream
}

type cDNSubscribeServer struct {
	grpc.ServerStream
}

func (x *cDNSubscribeServer) Send(m *SubscribeResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *cDNSubscribeServer) Recv() (*SubscribeRequest, error) {
	m := new(SubscribeRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// CDN_ServiceDesc is the grpc.ServiceDesc for CDN service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var CDN_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "api.CDN",
	HandlerType: (*CDNServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Publish",
			Handler:       _CDN_Publish_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "Subscribe",
			Handler:       _CDN_Subscribe_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "cdn.proto",
}
