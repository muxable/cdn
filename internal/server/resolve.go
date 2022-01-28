package server

import (
	"context"

	"github.com/muxable/cdn/api"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *CDNServer) Resolve(ctx context.Context, req *emptypb.Empty) (*api.ResolveResponse, error) {
	return &api.ResolveResponse{Cname: s.config.InboundAddress}, nil
}