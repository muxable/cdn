package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	"github.com/muxable/cdn/internal/server"
	"go.uber.org/zap"
)

func main() {
	size := flag.Int("size", 24, "number of nodes in the swarm to spawn")
	flag.Parse()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	undo := zap.ReplaceGlobals(logger)
	defer undo()

	for i := 0; i < *size; i++ {
		port := fmt.Sprintf("%d", i+50051)
		
		if i > 0 {
			// pick a random previous node to bootstrap from
			j := rand.Intn(i)
			bootstrap := fmt.Sprintf("127.0.0.1:%d", j+50051)
			go server.ServeCDN("127.0.0.1", port, &bootstrap)
		} else {
			go server.ServeCDN("127.0.0.1", port, nil)
		}
		time.Sleep(100 * time.Millisecond)
	}

	select{}
}
