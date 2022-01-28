package server

import (
	"crypto/sha256"
	"fmt"

	"github.com/muxable/cdn/api"
	"github.com/muxable/cdn/internal/signal"
	"github.com/muxable/cdn/internal/store"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

func (s *CDNServer) Publish(conn api.CDN_PublishServer) error {
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
			if err := conn.Send(&api.PublishResponse{
				Operation: &api.PublishResponse_Signal{Signal: signal},
			}); err != nil {
				zap.L().Error("failed to send signal", zap.Error(err))
				return
			}
		}
	}()

	peerConnection.OnTrack(func(tr *webrtc.TrackRemote, r *webrtc.RTPReceiver) {
		zap.L().Info("track received", zap.String("kind", tr.Kind().String()))

		go func() {
			buf := make([]byte, 1500)
			for {
				if _, _, err := r.Read(buf); err != nil {
					return
				}
			}
		}()

		// hex format this for ease of rendering and debugging.
		key := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s", tr.StreamID(), tr.ID(), tr.RID()))))

		// save the track in the local store.
		s.config.LocalStore.Put(key, &store.TrackRemote{TrackRemote: tr, Latency: 0}) // since this is the ingressor, the total latency is zero.

		// announce to the swarm that we have this track.
		if err := s.config.DHTStore.Put(key, s.config.InboundAddress); err != nil {
			zap.L().Error("failed to store track", zap.Error(err))
			return
		}

		if err := conn.Send(&api.PublishResponse{
			Operation: &api.PublishResponse_Track{
				Track: &api.Track{
					Id:          tr.ID(),
					StreamId:    tr.StreamID(),
					RtpStreamId: tr.RID(),
					Key:         key,
				},
			},
		}); err != nil {
			zap.L().Error("failed to send stream id", zap.Error(err))
		}
	})

	for {
		in, err := conn.Recv()
		if err != nil {
			zap.L().Error("failed to receive", zap.Error(err))
			return nil
		}
		if err := signaller.WriteSignal(in.Signal); err != nil {
			zap.L().Error("failed to write signal", zap.Error(err))
			return nil
		}
	}
}
