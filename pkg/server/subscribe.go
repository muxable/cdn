package server

import (
	"context"
	"errors"

	"github.com/muxable/cdn/api"
	"github.com/muxable/cdn/pkg/cdn"
	"github.com/muxable/signal/pkg/signal"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (s *CDNServer) relay(ctx context.Context, streamID string) error {
	// fetch the publisher address from firestore.
	snapshot, err := s.config.Firestore.Collection("streams").Doc(streamID).Get(ctx)
	if err != nil {
		return err
	}
	if !snapshot.Exists() || snapshot.Data()["publisher"] == nil {
		return errors.New("stream does not exist")
	}
	publisher := snapshot.Data()["publisher"].(string)

	if publisher == s.config.InboundAddress {
		return nil
	}

	// connect to the publisher.
	conn, err := grpc.Dial(publisher, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	client, err := cdn.NewClient(conn)
	if err != nil {
		return err
	}

	// subscribe to the publisher.
	publisherPeerConnection, err := client.Subscribe(streamID)
	if err != nil {
		return err
	}

	// add the publisher to the local store.
	s.config.LocalStore.AddPublisher(publisherPeerConnection)

	publisherPeerConnection.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
		if pcs == webrtc.PeerConnectionStateClosed {
			conn.Close()
		}
	})

	return nil
}

func (s *CDNServer) Subscribe(conn api.CDN_SubscribeServer) error {
	peerConnection, err := webrtc.NewPeerConnection(s.config.WebRTCConfiguration)
	if err != nil {
		return err
	}

	signaller := signal.NewSignaller(peerConnection)

	peerConnection.OnNegotiationNeeded(signaller.Renegotiate)

	go func() {
		for {
			signal, err := signaller.ReadSignal()
			if err != nil {
				zap.L().Error("failed to read signal", zap.Error(err))
				return
			}
			if err := conn.Send(&api.SubscribeResponse{Signal: signal}); err != nil {
				zap.L().Error("failed to send signal", zap.Error(err))
				return
			}
		}
	}()

	for {
		in, err := conn.Recv()
		if err != nil {
			zap.L().Error("failed to receive", zap.Error(err))
			return nil
		}

		switch operation := in.Operation.(type) {
		case *api.SubscribeRequest_Subscription_:
			// if the stream id is not linked on this server, subscribe to the publisher.
			s.streamMutex.Lock()
			if !s.linkedStreamIDs[operation.Subscription.StreamId] {
				if err := s.relay(context.Background(), operation.Subscription.StreamId); err != nil {
					zap.L().Error("failed to relay", zap.Error(err))
					s.streamMutex.Unlock()
					return nil
				}
				s.linkedStreamIDs[operation.Subscription.StreamId] = true
			}
			s.streamMutex.Unlock()

			go func() {
				for tl := range s.config.LocalStore.Subscribe(conn.Context(), operation.Subscription.StreamId) {
					rtpSender, err := peerConnection.AddTrack(tl)
					if err != nil {
						zap.L().Error("failed to add track", zap.Error(err))
						return
					}

					go func() {
						for {
							if _, _, err := rtpSender.ReadRTCP(); err != nil {
								return
							}
						}
					}()
				}
			}()

		case *api.SubscribeRequest_Signal:
			if err := signaller.WriteSignal(operation.Signal); err != nil {
				return err
			}
		}
	}
}
