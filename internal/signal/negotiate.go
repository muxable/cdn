package signal

import (
	"encoding/json"
	"io"

	"github.com/muxable/cdn/api"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

type Signaller struct {
	pc     *webrtc.PeerConnection
	readCh chan *api.Signal
}

func (s *Signaller) ReadSignal() (*api.Signal, error) {
	signal, ok := <-s.readCh
	if !ok {
		return nil, io.EOF
	}
	return signal, nil
}

func (s *Signaller) WriteSignal(signal *api.Signal) error {
	switch payload := signal.Payload.(type) {
	case *api.Signal_OfferSdp:
		if err := s.pc.SetRemoteDescription(webrtc.SessionDescription{
			SDP:  payload.OfferSdp,
			Type: webrtc.SDPTypeOffer,
		}); err != nil {
			return err
		}
		answer, err := s.pc.CreateAnswer(nil)
		if err != nil {
			return err
		}

		if err := s.pc.SetLocalDescription(answer); err != nil {
			return err
		}

		s.readCh <- &api.Signal{Payload: &api.Signal_AnswerSdp{AnswerSdp: answer.SDP}}
	case *api.Signal_AnswerSdp:
		if err := s.pc.SetRemoteDescription(webrtc.SessionDescription{
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

		if err := s.pc.AddICECandidate(candidate); err != nil {
			return err
		}
	}
	return nil
}

func Negotiate(pc *webrtc.PeerConnection) *Signaller {
	s := &Signaller{
		pc:     pc,
		readCh: make(chan *api.Signal),
	}

	pc.OnNegotiationNeeded(func() {
		offer, err := pc.CreateOffer(nil)
		if err != nil {
			zap.L().Error("failed to create offer", zap.Error(err))
			return
		}

		if err := pc.SetLocalDescription(offer); err != nil {
			zap.L().Error("failed to set local description", zap.Error(err))
			return
		}

		s.readCh <- &api.Signal{Payload: &api.Signal_OfferSdp{OfferSdp: offer.SDP}}
	})
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		trickle, err := json.Marshal(candidate.ToJSON())
		if err != nil {
			return
		}

		s.readCh <- &api.Signal{Payload: &api.Signal_Trickle{Trickle: string(trickle)}}
	})
	return s
}
