package server

import (
	"encoding/json"
	"fmt"

	"github.com/muxable/cdn/api"
	"github.com/muxable/cdn/internal/store"
	"github.com/muxable/cdn/pkg/cdn"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Source struct {
	*webrtc.PeerConnection
	*webrtc.TrackRemote
}

type CDNServer struct {
	api.UnimplementedCDNServer
	config webrtc.Configuration

	publisherStore store.DHTStore
	leechResolver  store.NodeResolver
}

func NewCDNServer(config webrtc.Configuration, store store.TrackStore) *CDNServer {
	return &CDNServer{
		config: config,
		store:  store,
	}
}

func (s *CDNServer) Publish(conn api.CDN_PublishServer) error {
	peerConnection, err := webrtc.NewPeerConnection(s.config)
	if err != nil {
		return err
	}

	peerConnection.OnNegotiationNeeded(func() {
		offer, err := peerConnection.CreateOffer(nil)
		if err != nil {
			zap.L().Error("failed to create offer", zap.Error(err))
			return
		}

		if err := peerConnection.SetLocalDescription(offer); err != nil {
			zap.L().Error("failed to set local description", zap.Error(err))
			return
		}

		if err := conn.Send(&api.PublishResponse{
			Operation: &api.PublishResponse_Signal{
				Signal: &api.Signal{
					Payload: &api.Signal_OfferSdp{
						OfferSdp: offer.SDP,
					},
				},
			},
		}); err != nil {
			zap.L().Error("failed to send offer", zap.Error(err))
		}
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		trickle, err := json.Marshal(candidate.ToJSON())
		if err != nil {
			zap.L().Error("failed to marshal candidate", zap.Error(err))
			return
		}

		if err := conn.Send(&api.PublishResponse{
			Operation: &api.PublishResponse_Signal{
				Signal: &api.Signal{
					Payload: &api.Signal_Trickle{Trickle: string(trickle)},
				},
			},
		}); err != nil {
			zap.L().Error("failed to send candidate", zap.Error(err))
		}
	})

	peerConnection.OnTrack(func(tr *webrtc.TrackRemote, r *webrtc.RTPReceiver) {
		go func() {
			buf := make([]byte, 1500)
			for {
				if _, _, err := r.Read(buf); err != nil {
					return
				}
			}
		}()

		// announce to the swarm that we have this track.
		cname, err := store.GetLocalAddress()
		if err != nil {
			zap.L().Error("failed to get local address", zap.Error(err))
			return
		}

		key := fmt.Sprintf("%s:%s:%s", tr.StreamID(), tr.ID(), tr.RID())
		if err := s.publisherStore.Put(key, cname); err != nil {
			zap.L().Error("failed to store track", zap.Error(err))
			return
		}

		if err := conn.Send(&api.PublishResponse{
			Operation: &api.PublishResponse_StreamId{
				StreamId: key,
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

		switch payload := in.Signal.Payload.(type) {
		case *api.Signal_OfferSdp:
			if err := peerConnection.SetRemoteDescription(webrtc.SessionDescription{
				SDP:  payload.OfferSdp,
				Type: webrtc.SDPTypeOffer,
			}); err != nil {
				return err
			}
			answer, err := peerConnection.CreateAnswer(nil)
			if err != nil {
				return err
			}

			if err := peerConnection.SetLocalDescription(answer); err != nil {
				return err
			}

			if err := conn.Send(&api.PublishResponse{
				Operation: &api.PublishResponse_Signal{
					Signal: &api.Signal{
						Payload: &api.Signal_AnswerSdp{AnswerSdp: answer.SDP},
					},
				},
			}); err != nil {
				return err
			}

		case *api.Signal_AnswerSdp:
			if err := peerConnection.SetRemoteDescription(webrtc.SessionDescription{
				SDP:  payload.AnswerSdp,
				Type: webrtc.SDPTypeAnswer,
			}); err != nil {
				return err
			}

		case *api.Signal_Trickle:
			candidate := webrtc.ICECandidateInit{}
			if err := json.Unmarshal([]byte(payload.Trickle), &candidate); err != nil {
				return err
			}

			if err := peerConnection.AddICECandidate(candidate); err != nil {
				return err
			}
		}
	}
}

func (s *CDNServer) Subscribe(conn api.CDN_SubscribeServer) error {
	peerConnection, err := webrtc.NewPeerConnection(s.config)
	if err != nil {
		return err
	}

	peerConnection.OnNegotiationNeeded(func() {
		offer, err := peerConnection.CreateOffer(nil)
		if err != nil {
			zap.L().Error("failed to create offer", zap.Error(err))
			return
		}

		if err := peerConnection.SetLocalDescription(offer); err != nil {
			zap.L().Error("failed to set local description", zap.Error(err))
			return
		}

		if err := conn.Send(&api.SubscribeResponse{
			Signal: &api.Signal{
				Payload: &api.Signal_OfferSdp{
					OfferSdp: offer.SDP,
				},
			},
		}); err != nil {
			zap.L().Error("failed to send offer", zap.Error(err))
		}
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		trickle, err := json.Marshal(candidate.ToJSON())
		if err != nil {
			zap.L().Error("failed to marshal candidate", zap.Error(err))
			return
		}

		if err := conn.Send(&api.SubscribeResponse{
			Signal: &api.Signal{
				Payload: &api.Signal_Trickle{Trickle: string(trickle)},
			},
		}); err != nil {
			zap.L().Error("failed to send candidate", zap.Error(err))
		}
	})

	for {
		in, err := conn.Recv()
		if err != nil {
			zap.L().Error("failed to receive", zap.Error(err))
			return nil
		}

		switch operation := in.Operation.(type) {
		case *api.SubscribeRequest_StreamId:
			if localMulticaster.HasTrack(operation.StreamId) {
				tl, err := localMulticaster.GetTrack(operation.StreamId)
				if err != nil {
					return err
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
			}
			publisher, err := s.publisherStore.Get(operation.StreamId)
			if err != nil {
				return err
			}

			leechFrom, err := s.leechResolver.Resolve(publisher, operation.StreamId)
			if err != nil {
				return err
			}

			// create a new peer connection and send a new subscribe request to the leech target.
			leechConn, err := grpc.Dial(leechFrom.LeecherAddr, grpc.WithTransportCredentials(insecure.NewTransportCredentials()))
			if err != nil {
				return err
			}
			client := cdn.NewClient(conn.Context(), leechConn)
			tr, err := client.Subscribe(operation.StreamId)
			if err != nil {
				return err
			}

			// forward the remote track to the subscriber.
			tl, err := webrtc.NewTrackLocalStaticRTP(tr.Codec().RTPCodecCapability, tr.ID(), tr.StreamID())
			if err != nil {
				return err
			}

			go func() {
				for {
					p, _, err := tr.ReadRTP()
					if err != nil {
						return
					}
					if err := tl.WriteRTP(p); err != nil {
						return
					}
				}
			}()

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

		case *api.SubscribeRequest_Signal:
			switch payload := operation.Signal.Payload.(type) {
			case *api.Signal_OfferSdp:
				if err := peerConnection.SetRemoteDescription(webrtc.SessionDescription{
					SDP:  payload.OfferSdp,
					Type: webrtc.SDPTypeOffer,
				}); err != nil {
					return err
				}
				answer, err := peerConnection.CreateAnswer(nil)
				if err != nil {
					return err
				}

				if err := peerConnection.SetLocalDescription(answer); err != nil {
					return err
				}

				if err := conn.Send(&api.SubscribeResponse{
					Signal: &api.Signal{
						Payload: &api.Signal_AnswerSdp{AnswerSdp: answer.SDP},
					},
				}); err != nil {
					return err
				}

			case *api.Signal_AnswerSdp:
				if err := peerConnection.SetRemoteDescription(webrtc.SessionDescription{
					SDP:  payload.AnswerSdp,
					Type: webrtc.SDPTypeAnswer,
				}); err != nil {
					return err
				}

			case *api.Signal_Trickle:
				candidate := webrtc.ICECandidateInit{}
				if err := json.Unmarshal([]byte(payload.Trickle), &candidate); err != nil {
					return err
				}

				if err := peerConnection.AddICECandidate(candidate); err != nil {
					return err
				}
			}
		}
	}
}
