package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/muxable/cdn/api"
	"github.com/muxable/cdn/internal/store"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type Subscriber struct {
	cname   string
	latency time.Duration
}

type Configuration struct {
	WebRTCConfiguration webrtc.Configuration
	DHTStore            *store.DHTStore
	LocalStore          *store.LocalTrackStore
	InboundAddress      string
}

type CDNServer struct {
	api.UnimplementedCDNServer
	config Configuration

	subscriberLock sync.Mutex
	subscribers    map[string][]*Subscriber
}

func NewCDNServer(config Configuration) *CDNServer {
	return &CDNServer{
		config:      config,
		subscribers: make(map[string][]*Subscriber),
	}
}

func ServeCDN(host, port string, probe string) error {
	addr := fmt.Sprintf("%s:%s", host, port)

	grpcConn, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	dhtConn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}

	udpAddr, err := net.ResolveUDPAddr("udp", "localhost:" + port)
	if err != nil {
		return err
	}
	
	bootstrapAddrs := []*net.UDPAddr{udpAddr}
	if probe != "" {
		probeAddr, err := store.Probe(context.Background(), probe)
		if err != nil {
			zap.L().Warn("failed to probe", zap.String("probe", probe), zap.Error(err))
		} else {
			bootstrapAddrs = append(bootstrapAddrs, probeAddr)
		}
	}

	dht, err := store.NewDHTStore(context.Background(), dhtConn, bootstrapAddrs...)
	if err != nil {
		return err
	}

	local := store.NewLocalTrackStore()

	inboundAddr, err := store.GetLocalAddress()
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()

	api.RegisterCDNServer(grpcServer, NewCDNServer(Configuration{
		WebRTCConfiguration: webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{URLs: []string{"stun:stun.l.google.com:19302"}},
			},
		},
		DHTStore:       dht,
		LocalStore:     local,
		InboundAddress: fmt.Sprintf("%s:%s", inboundAddr, port),
	}))
	grpc_health_v1.RegisterHealthServer(grpcServer, health.NewServer())

	zap.L().Info("starting cdn server", zap.String("addr", addr))

	return grpcServer.Serve(grpcConn)
}
