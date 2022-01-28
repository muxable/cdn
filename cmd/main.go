package main

import (
	"flag"
	"os"

	"github.com/blendle/zapdriver"
	"github.com/muxable/cdn/internal/server"
	"go.uber.org/zap"
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

	if err := server.ServeCDN("0.0.0.0", port, bootstrap); err != nil {
		panic(err)
	}
}
