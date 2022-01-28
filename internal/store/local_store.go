package store

import (
	"sync"
	"time"

	"github.com/pion/rtp"
	"github.com/pion/rtpio/pkg/rtpio"
	"github.com/pion/webrtc/v3"
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

	Latency time.Duration
	Trace   []string
}

type TrackLocal struct {
	*webrtc.TrackLocalStaticRTP

	Latency time.Duration
	Trace   []string
}

// LocalTrackStore is a track store that stores tracks in memory.
type LocalTrackStore struct {
	sync.RWMutex

	tracks map[string]*TrackRemote
}

func NewLocalTrackStore() *LocalTrackStore {
	return &LocalTrackStore{
		tracks: make(map[string]*TrackRemote),
	}
}

func (s *LocalTrackStore) GetLatency(key string) (time.Duration, error) {
	s.RLock()
	defer s.RUnlock()

	track, ok := s.tracks[key]
	if !ok {
		return 0, ErrNotFound
	}
	return track.Latency, nil
}

func (s *LocalTrackStore) Get(key string) (*TrackLocal, error) {
	s.RLock()
	defer s.RUnlock()

	tr, ok := s.tracks[key]
	if !ok {
		return nil, ErrNotFound
	}
	tl, err := webrtc.NewTrackLocalStaticRTP(tr.Codec().RTPCodecCapability, tr.ID(), tr.StreamID(), webrtc.WithRTPStreamID(tr.RID()))
	if err != nil {
		return nil, err
	}
	tr.multicaster.WriteTo(tl)
	return &TrackLocal{
		TrackLocalStaticRTP: tl,
		Latency:             tr.Latency,
		Trace:               tr.Trace,
	}, nil
}

func (s *LocalTrackStore) Put(key string, track *TrackRemote) error {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.tracks[key]; ok {
		return ErrAlreadyExists
	}
	track.multicaster = NewMulticaster(&TrackRemoteReader{TrackRemote: track.TrackRemote})
	s.tracks[key] = track
	return nil
}
