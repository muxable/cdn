package store

import (
	"context"
	"net"

	"github.com/muxable/cdn/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

func Probe(ctx context.Context, addr string) (*net.UDPAddr, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	client := api.NewCDNClient(conn)
	response, err := client.Resolve(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	resolved, err := net.ResolveUDPAddr("udp", response.Cname)
	if err != nil {
		return nil, err
	}
	return resolved, nil
}