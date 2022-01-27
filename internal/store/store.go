package store

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"net"
	"time"

	"github.com/anacrolix/dht/v2"
	"github.com/anacrolix/dht/v2/bep44"
	"github.com/anacrolix/dht/v2/exts/getput"
	"github.com/anacrolix/dht/v2/krpc"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

type TrackStore interface {
	Get(key string) (webrtc.TrackLocal, error)
	Put(key string, value *webrtc.TrackRemote) error
	Del(key string) error
}

var ErrNotFound = errors.New("not found")
var ErrAlreadyExists = errors.New("already exists")

func pipe(tr *webrtc.TrackRemote) (webrtc.TrackLocal, error) {
	return nil, nil
	// tl, err := webrtc.NewTrackLocalStaticRTP(tr.Codec().RTPCodecCapability, tr.ID(), tr.StreamID())
	// if err != nil {
	// 	return nil, err
	// }
	// go func() {
	// 	for {
	// 		pkt, _, err := tr.ReadRTP()
	// 		if err != nil {
	// 			return
	// 		}
	// 		if err := tl.WriteRTP(pkt); err != nil {
	// 			return
	// 		}
	// 	}
	// }()
	// return tl, nil
}

// DHTStore is a TrackStore that uses a DHT to store tracks if they don't exist on a local store.
type DHTStore struct {
	*dht.Server
	ctx context.Context
}

func NewDHTTrackStore(ctx context.Context, conn net.PacketConn, bootstrap net.Addr) (*DHTStore, error) {
	server, err := dht.NewServer(&dht.ServerConfig{
		NoSecurity: true,
		Conn:       conn,
		StartingNodes: func() ([]dht.Addr, error) {
			if bootstrap == nil {
				zap.L().Warn("no bootstrap node provided")
				return nil, nil
			}
			return []dht.Addr{dht.NewAddr(bootstrap)}, nil
		},
		DefaultWant: []krpc.Want{krpc.WantNodes, krpc.WantNodes6},
		Store:       bep44.NewMemory(),
		Exp:         2 * time.Hour,
		SendLimiter: dht.DefaultSendLimiter,
	})
	if err != nil {
		return nil, err
	}
	if _, err := server.Bootstrap(); err != nil {
		return nil, err
	}
	return &DHTStore{
		ctx:    ctx,
		Server: server,
	}, nil
}

func signature(key string) ed25519.PrivateKey {
	seed := sha256.Sum256([]byte(key))
	priv := ed25519.NewKeyFromSeed(seed[:])
	return priv
}

// Get returns the stored key.
func (s *DHTStore) Get(key string) (string, error) {
	item, err := bep44.NewItem(nil, nil, 0, 0, signature(key))
	if err != nil {
		return "", err
	}
	res, _, err := getput.Get(s.ctx, item.Target(), s.Server, nil, nil)
	if err != nil {
		return "", err
	}

	// this is the publisher's address.
	return res.V.(string), nil
}

func (s *DHTStore) Put(key, value string) error {
	item, err := bep44.NewItem(value, nil, 0, 0, signature(key))
	if err != nil {
		return err
	}

	_, err = getput.Put(context.Background(), item.Target(), s.Server, item.ToPut())
	return err
}

func (s *DHTStore) Del(key string) error {
	// TODO: implement
	return nil
}
