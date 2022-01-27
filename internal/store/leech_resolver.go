package store

import (
	"context"
	"time"

	"github.com/muxable/cdn/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Result struct {
	LeecherAddr        string
	IncrementalLatency time.Duration
	PublisherLatency   time.Duration
}

func ResolveLeech(ctx context.Context, publisherAddr, streamId string) (*Result, error) {
	return traverse(ctx, publisherAddr, streamId, 0)
}

func traverse(ctx context.Context, publisherAddr, streamId string, totalLatency time.Duration) (*Result, error) {
	conn, err := grpc.Dial(publisherAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := api.NewCDNClient(conn)

	sendTs := time.Now()
	response, err := client.Traverse(ctx, &api.TraverseRequest{
		StreamId: streamId,
	})
	recvTs := time.Now()
	if err != nil {
		return nil, err
	}

	// check if we can subscribe to this node.
	if len(response.Subscribers) < int(response.RequestedMaxSubscribers) {
		return &Result{
			LeecherAddr:        publisherAddr,
			IncrementalLatency: recvTs.Sub(sendTs),
			PublisherLatency:   totalLatency,
		}, nil
	}

	// otherwise, we need to traverse to each of the subscribers, choosing the one with the lowest total latency.

}
