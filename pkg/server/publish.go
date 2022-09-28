package server

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"
	"github.com/muxable/cdn/api"
	"github.com/muxable/cdn/internal/store"
	"github.com/muxable/signal/pkg/signal"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

func (s *CDNServer) Publish(conn api.CDN_PublishServer) error {
	peerConnection, err := webrtc.NewPeerConnection(s.config.WebRTCConfiguration)
	if err != nil {
		return err
	}
	defer peerConnection.Close()

	signaller := signal.NewSignaller(peerConnection)

	peerConnection.OnNegotiationNeeded(signaller.Renegotiate)

	go func() {
		for {
			signal, err := signaller.ReadSignal()
			if err != nil {
				zap.L().Error("failed to read signal", zap.Error(err))
				return
			}
			if err := conn.Send(&api.PublishResponse{Signal: signal}); err != nil {
				zap.L().Error("failed to send signal", zap.Error(err))
				return
			}
		}
	}()

	peerConnection.OnTrack(func(tr *webrtc.TrackRemote, r *webrtc.RTPReceiver) {
		zap.L().Info("track received", zap.String("kind", tr.Kind().String()))

		ref := s.config.Firestore.Collection("streams").Doc(tr.StreamID())

		// declare us as the publisher of this stream.
		if err := s.config.Firestore.RunTransaction(conn.Context(), func(ctx context.Context, tx *firestore.Transaction) error {
			doc, err := tx.Get(ref)
			if err != nil && doc == nil {
				return err
			}
			if doc.Exists() && doc.Data()["publisher"] != s.config.InboundAddress {
				return errors.New("stream already published")
			}
			return tx.Set(ref, map[string]interface{}{
				"publisher": s.config.InboundAddress,
				"trackIds":  firestore.ArrayUnion(tr.ID()),
				"updatedAt": firestore.ServerTimestamp,
			}, firestore.MergeAll)
		}); err != nil {
			zap.L().Error("failed to declare publisher", zap.Error(err))
			return
		}

		s.streamMutex.Lock()
		s.linkedStreamIDs[tr.StreamID()] = true
		s.streamMutex.Unlock()

		go func() {
			buf := make([]byte, 1500)
			for {
				if _, _, err := r.Read(buf); err != nil {
					// remove the track id, use bg context to avoid cancellation.
					if _, err := ref.Update(context.Background(), []firestore.Update{
						{Path: "trackIds", Value: firestore.ArrayRemove(tr.ID())},
						{Path: "updatedAt", Value: firestore.ServerTimestamp},
					}); err != nil {
						zap.L().Error("failed to declare publisher", zap.Error(err))
					}
					return
				}
			}
		}()

		// save the track in the local store.
		if err := s.config.LocalStore.AddTrack(&store.TrackRemote{TrackRemote: tr}); err != nil {
			zap.L().Error("failed to add track", zap.Error(err))
			return
		}
	})

	for {
		in, err := conn.Recv()
		if err != nil {
			return nil
		}
		if err := signaller.WriteSignal(in.Signal); err != nil {
			zap.L().Error("failed to write signal", zap.Error(err))
			return nil
		}
	}
}
