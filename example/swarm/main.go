package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/muxable/cdn/pkg/server"
	"go.uber.org/zap"
)

func main() {
	size := flag.Int("size", 2, "number of nodes in the swarm to spawn")
	flag.Parse()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	undo := zap.ReplaceGlobals(logger)
	defer undo()

	for i := 0; i < *size; i++ {
		go server.ServeCDN(fmt.Sprintf("127.0.0.1:%d", i+50051))
		// in order to guarantee a connected graph, we need to wait a bit
		// to let each individual server start up.
		time.Sleep(1 * time.Second)
	}

	select{}
}
