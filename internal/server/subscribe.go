package server

import (
	"github.com/muxable/cdn/api"
	"github.com/muxable/cdn/internal/signal"
	"github.com/muxable/cdn/internal/store"
	"github.com/muxable/cdn/pkg/cdn"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (s *CDNServer) Subscribe(conn api.CDN_SubscribeServer) error {
	peerConnection, err := webrtc.NewPeerConnection(s.config.WebRTCConfiguration)
	if err != nil {
		return err
	}

	signaller := signal.Negotiate(peerConnection)

	go func() {
		for {
			signal, err := signaller.ReadSignal()
			if err != nil {
				zap.L().Error("failed to read signal", zap.Error(err))
				return
			}
			if err := conn.Send(&api.SubscribeResponse{
				Operation: &api.SubscribeResponse_Signal{Signal: signal},
			}); err != nil {
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
			key := operation.Subscription.Key
			tl, err := s.config.LocalStore.Get(key)
			if err != nil {
				if err != store.ErrNotFound {
					return err
				}
				publisher, err := s.config.DHTStore.Get(key)
				if err != nil {
					return err
				}

				leechFrom, err := store.ResolveLeech(conn.Context(), publisher, key)
				if err != nil {
					return err
				}

				// create a new peer connection and send a new subscribe request to the leech target.
				leechConn, err := grpc.Dial(leechFrom.LeecherAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
				if err != nil {
					return err
				}
				client, err := cdn.NewClient(leechConn)
				if err != nil {
					return err
				}

				tr, err := client.Subscribe(conn.Context(), key)
				if err != nil {
					return err
				}

				// put it in the local store.
				if err := s.config.LocalStore.Put(key, &store.TrackRemote{
					TrackRemote: tr.TrackRemote,
					Latency:     leechFrom.IncrementalLatency + leechFrom.PublisherLatency,
					Trace:       tr.Trace,
				}); err != nil {
					return err
				}

				// load our new track.
				tl, err = s.config.LocalStore.Get(key)
				if err != nil {
					return err
				}
			}

			rtpSender, err := peerConnection.AddTrack(tl)
			if err != nil {
				return err
			}

			go func() {
				for {
					if _, _, err := rtpSender.ReadRTCP(); err != nil {
						return
					}
				}
			}()

			// then notify the subscription on the signalling channel.
			if err := conn.Send(&api.SubscribeResponse{
				Operation: &api.SubscribeResponse_Track{
					Track: &api.Track{
						Id:          tl.ID(),
						StreamId:    tl.StreamID(),
						RtpStreamId: tl.RID(),
						Key:         key,
						Trace:       append(tl.Trace, s.config.InboundAddress),
					},
				},
			}); err != nil {
				return err
			}

		case *api.SubscribeRequest_Signal:
			if err := signaller.WriteSignal(operation.Signal); err != nil {
				return err
			}
		}
	}
}
