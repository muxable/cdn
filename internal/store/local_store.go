package store

import (
	"sync"

	"github.com/pion/webrtc/v3"
)

// LocalTrackStore is a track store that stores tracks in memory.
type LocalTrackStore struct {
	sync.RWMutex

	tracks map[string]*webrtc.TrackRemote
}

func NewLocalTrackStore() *LocalTrackStore {
	return &LocalTrackStore{
		tracks: make(map[string]*webrtc.TrackRemote),
	}
}

func (s *LocalTrackStore) Get(key string) (webrtc.TrackLocal, error) {
	s.RLock()
	defer s.RUnlock()

	tr, ok := s.tracks[key]
	if !ok {
		return nil, ErrNotFound
	}
	return pipe(tr)
}

func (s *LocalTrackStore) Put(key string, tr *webrtc.TrackRemote) error {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.tracks[key]; ok {
		return ErrAlreadyExists
	}
	s.tracks[key] = tr
	return nil
}
