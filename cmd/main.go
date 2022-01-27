package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/anacrolix/dht/v2/bep44"
	"github.com/anacrolix/dht/v2/exts/getput"
	"github.com/blendle/zapdriver"
	"github.com/muxable/cdn/api"
	"github.com/muxable/cdn/internal/server"
	"github.com/muxable/cdn/internal/store"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func logger() (*zap.Logger, error) {
	if os.Getenv("APP_ENV") == "production" {
		return zapdriver.NewProduction()
	} else {
		return zap.NewDevelopment()
	}
}

func main() {
	bootstrap := flag.String("bootstrap", "", "bootstrap node")
	doput := flag.Bool("doput", false, "do put")
	doget := flag.Bool("doget", false, "do get")
	flag.Parse()

	logger, err := logger()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	undo := zap.ReplaceGlobals(logger)
	defer undo()

	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}

	addr := fmt.Sprintf(":%s", port)

	grpcConn, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	bootstrapAddr, err := net.ResolveUDPAddr("udp", *bootstrap)
	if err != nil {
		panic(err)
	}

	dhtConn, err := net.ListenPacket("udp", addr)
	if err != nil {
		panic(err)
	}

	trackStore, err := store.NewDHTTrackStore(context.Background(), dhtConn, bootstrapAddr)
	if err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer()

	api.RegisterCDNServer(grpcServer, server.NewCDNServer(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}, trackStore))
	grpc_health_v1.RegisterHealthServer(grpcServer, health.NewServer())

	zap.L().Info("starting cdn server", zap.String("addr", addr))

	seed := sha256.Sum256([]byte("qqq"))
	priv := ed25519.NewKeyFromSeed(seed[:])

	if *doput {
		item, err := bep44.NewItem("Hello World!", nil, 0, 0, priv)
		if err != nil {
			panic(err)
		}

		// send get request to s2, we need a write token to put data
		stats, err := getput.Put(context.Background(), item.Target(), trackStore.Server, item.ToPut())
		if err != nil {
			panic(err)
		}
		fmt.Printf("stats: %+v\n", stats)

		log.Printf("%x", item.Target())
	} else if *doget {
		item, err := bep44.NewItem(nil, nil, 0, 0, priv)
		if err != nil {
			panic(err)
		}

		res, stats, err := getput.Get(context.Background(), item.Target(), trackStore.Server, nil, nil)
		if err != nil {
			panic(err)
		}
		fmt.Printf("stats: %+v\n", stats)
		fmt.Printf("res: %+v\n", res)
	}

	if err := grpcServer.Serve(grpcConn); err != nil {
		panic(err)
	}
}
