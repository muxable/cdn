package main

import (
	"context"
	"flag"
	"log"

	"github.com/muxable/cdn/pkg/cdn"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := flag.String("addr", "", "destination address")
	key := flag.String("key", "", "key")
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

	tr, err := client.Subscribe(context.Background(), *key)
	if err != nil {
		panic(err)
	}

	log.Printf("%+v", tr.Trace)

	for {
		p, _, err := tr.ReadRTP()
		if err != nil {
			panic(err)
		}
		log.Printf("received %d bytes", p.MarshalSize())
	}
}