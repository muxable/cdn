package store

import (
	"context"
	"sync"

	"github.com/pion/rtp"
	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

type TrackRemoteReader struct {
	*webrtc.TrackRemote
}

// ReadRTP reads an RTP packet from the track.
func (r *TrackRemoteReader) ReadRTP() (*rtp.Packet, error) {
	p, _, err := r.TrackRemote.ReadRTP()
	return p, err
}

var _ rtpio.RTPReader = (*TrackRemoteReader)(nil)

type Multicaster struct {
	sync.Mutex

	sink    rtpio.RTPReader
	sources []rtpio.RTPWriter
}

func NewMulticaster(sink rtpio.RTPReader) *Multicaster {
	m := &Multicaster{sink: sink}

	go func() {
		for {
			p, err := sink.ReadRTP()
			if err != nil {
				return
			}

			m.Lock()
			for _, source := range m.sources {
				if err := source.WriteRTP(p); err != nil {
					continue
				}
			}
			m.Unlock()
		}
	}()

	return m
}

func (m *Multicaster) WriteTo(w rtpio.RTPWriter) {
	m.Lock()
	defer m.Unlock()

	m.sources = append(m.sources, w)
}

type TrackRemote struct {
	*webrtc.TrackRemote
	multicaster *Multicaster

	Trace   []string
}

type TrackLocal struct {
	*webrtc.TrackLocalStaticRTP

	Trace   []string
}

type Subscription struct {
	ch chan *TrackLocal
	StreamID string
}

// LocalTrackStore is a track store that stores tracks in memory.
type LocalTrackStore struct {
	sync.RWMutex

	tracks        []*TrackRemote
	subscriptions []*Subscription
}

func NewLocalTrackStore() *LocalTrackStore {
	return &LocalTrackStore{}
}

func (s *LocalTrackStore) Subscribe(ctx context.Context, streamID string) chan *TrackLocal {
	s.RLock()
	defer s.RUnlock()

	ch := make(chan *TrackLocal)

	for _, tr := range s.tracks {
		if tr.StreamID() == streamID {
			tl, err := webrtc.NewTrackLocalStaticRTP(tr.Codec().RTPCodecCapability, tr.ID(), tr.StreamID(), webrtc.WithRTPStreamID(tr.RID()))
			if err != nil {
				continue
			}
			tr.multicaster.WriteTo(tl)
			go func ()  {
				ch <- &TrackLocal{TrackLocalStaticRTP: tl, Trace: tr.Trace}
			}()
		}
	}
	s.subscriptions = append(s.subscriptions, &Subscription{ch: ch, StreamID: streamID})
	go func() {
		<-ctx.Done()
		s.Lock()
		defer s.Unlock()
		for i, sub := range s.subscriptions {
			if sub.ch == ch {
				s.subscriptions = append(s.subscriptions[:i], s.subscriptions[i+1:]...)
			}
		}
		close(ch)
	}()
	return ch
}

func (s *LocalTrackStore) AddTrack(track *TrackRemote) error {
	s.Lock()
	defer s.Unlock()

	track.multicaster = NewMulticaster(&TrackRemoteReader{TrackRemote: track.TrackRemote})
	for _, sub := range s.subscriptions {
		if sub.StreamID == track.StreamID() {
			tl, err := webrtc.NewTrackLocalStaticRTP(track.Codec().RTPCodecCapability, track.ID(), track.StreamID(), webrtc.WithRTPStreamID(track.RID()))
			if err != nil {
				return err
			}
			track.multicaster.WriteTo(tl)
			sub.ch <- &TrackLocal{TrackLocalStaticRTP: tl, Trace: track.Trace}
		}
	}
	s.tracks = append(s.tracks, track)
	return nil
}

func (s *LocalTrackStore) AddPublisher(pc *webrtc.PeerConnection) {
	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		if err := s.AddTrack(&TrackRemote{TrackRemote: track}); err != nil {
			zap.L().Error("failed to add track", zap.Error(err))
			return
		}
	})
}