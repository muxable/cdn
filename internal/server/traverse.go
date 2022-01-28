package server

import (
	"context"

	"github.com/muxable/cdn/api"
	"google.golang.org/protobuf/types/known/durationpb"
)

func (s *CDNServer) Traverse(ctx context.Context, req *api.TraverseRequest) (*api.TraverseResponse, error) {
	// verify that we have this stream.
	inboundLatency, err := s.config.LocalStore.GetLatency(req.StreamId)
	if err != nil {
		return nil, err
	}

	// list the subscribers.
	response := &api.TraverseResponse{RequestedMaxSubscribers: 10}  // TODO: dynamically adjust this based on bandwidth.

	s.subscriberLock.Lock()
	for _, subscriber := range s.subscribers[req.StreamId] {
		response.Subscribers = append(response.Subscribers, &api.TraverseResponse_Subscriber{
			Cname: subscriber.cname,
			Latency: durationpb.New(subscriber.latency + inboundLatency),
		})
	}
	s.subscriberLock.Unlock()
	
	return response, nil
}