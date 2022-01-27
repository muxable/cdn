package cdn

import (
	"context"

	"github.com/pion/webrtc/v3"
	"google.golang.org/grpc"
)

type Client struct {
	ctx      context.Context
	grpcConn *grpc.ClientConn
}

func NewClient(ctx context.Context, conn *grpc.ClientConn) (*Client, error) {
	return &Client{
		ctx:      ctx,
		grpcConn: conn,
	}, nil
}

func (c *Client) Publish(tl webrtc.TrackLocal) (key string, err error) {
	return "", nil
}

func (c *Client) Subscribe(key string) (tr *webrtc.TrackRemote, err error) {
	return nil, nil
}
