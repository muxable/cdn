package main

import (
	"flag"
	"log"

	"github.com/muxable/cdn/pkg/cdn"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := flag.String("addr", "", "destination address")
	flag.Parse()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	undo := zap.ReplaceGlobals(logger)
	defer undo()

	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	client, err := cdn.NewClient(conn)
	if err != nil {
		panic(err)
	}

	pc, err := client.Subscribe("pion")
	if err != nil {
		panic(err)
	}

	pc.OnTrack(func(tr *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		for {
			p, _, err := tr.ReadRTP()
			if err != nil {
				panic(err)
			}
			log.Printf("received %d bytes", p.MarshalSize())
		}
	})

	select{}
}