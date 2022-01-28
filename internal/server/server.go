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

func ServeCDN(host, port string, bootstrap *string) error {
	addr := fmt.Sprintf("%s:%s", host, port)

	grpcConn, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	var bootstrapAddr *net.UDPAddr
	if bootstrap != nil {
		bootstrapAddr, err = net.ResolveUDPAddr("udp", *bootstrap)
		if err != nil {
			panic(err)
		}
	}

	dhtConn, err := net.ListenPacket("udp", addr)
	if err != nil {
		panic(err)
	}

	dht, err := store.NewDHTStore(context.Background(), dhtConn, bootstrapAddr)
	if err != nil {
		panic(err)
	}

	local := store.NewLocalTrackStore()

	inboundAddr, err := store.GetLocalAddress()
	if err != nil {
		panic(err)
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