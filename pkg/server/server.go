package server

import (
	"context"
	"net"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/muxable/cdn/api"
	"github.com/muxable/cdn/internal/store"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	firebase "firebase.google.com/go/v4"
)

type Subscriber struct {
	cname   string
	latency time.Duration
}

type Configuration struct {
	WebRTCConfiguration webrtc.Configuration
	LocalStore          *store.LocalTrackStore
	Firestore           *firestore.Client
	InboundAddress      string
}

type CDNServer struct {
	api.UnimplementedCDNServer
	config Configuration

	linkedStreamIDs map[string]bool
	streamMutex     sync.Mutex
}

func NewCDNServer(config Configuration) *CDNServer {
	return &CDNServer{
		config:      config,
		linkedStreamIDs: make(map[string]bool),
	}
}

func ServeCDN(addr string) error {
	grpcConn, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	local := store.NewLocalTrackStore()

	app, err := firebase.NewApp(context.Background(), &firebase.Config{ProjectID: "rtirl-a1d7f"})
	if err != nil {
		return err
	}
	client, err := app.Firestore(context.Background())
	if err != nil {
		return err
	}
	defer client.Close()

	grpcServer := grpc.NewServer()

	api.RegisterCDNServer(grpcServer, NewCDNServer(Configuration{
		WebRTCConfiguration: webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{URLs: []string{"stun:stun.l.google.com:19302"}},
			},
		},
		Firestore:      client,
		LocalStore:     local,
		InboundAddress: addr,
	}))
	grpc_health_v1.RegisterHealthServer(grpcServer, health.NewServer())

	zap.L().Info("starting cdn server", zap.String("addr", addr))

	return grpcServer.Serve(grpcConn)
}
