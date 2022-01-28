package store

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"log"
	"net"
	"time"

	"github.com/anacrolix/dht/v2"
	"github.com/anacrolix/dht/v2/bep44"
	"github.com/anacrolix/dht/v2/krpc"
	"github.com/muxable/cdn/internal/store/getput"
	"go.uber.org/zap"
)

type DHTStore struct {
	*dht.Server
	ctx context.Context
}

func NewDHTStore(ctx context.Context, conn net.PacketConn, bootstrap *net.UDPAddr) (*DHTStore, error) {
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
	if bootstrap != nil {
		stats, err := server.Bootstrap()
		log.Printf("%#v", stats)
		if err != nil {
			return nil, err
		}
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				stats := server.Stats()
				zap.L().Debug("dht stats", zap.Any("stats", stats))
			}
		}
	}()
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
	res, err := getput.Get(s.ctx, item.Target(), s.Server)
	if err != nil {
		return "", err
	}

	// this is the publisher's address.
	return res, nil
}

func (s *DHTStore) Put(key, value string) error {
	item, err := bep44.NewItem(value, nil, 0, 0, signature(key))
	if err != nil {
		return err
	}
	return getput.Put(s.ctx, item.Target(), s.Server, item.ToPut())
}

func (s *DHTStore) Del(key string) error {
	// TODO: implement
	return nil
}