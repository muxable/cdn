package main

import (
	"context"
	"flag"
	"io"
	"os"
	"time"

	"github.com/muxable/cdn/pkg/cdn"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
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

	tl, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video", "pion")
	if err != nil {
		panic(err)
	}

	go func() {
		key, err := client.Publish(context.Background(), tl)
		if err != nil {
			panic(err)
		}

		zap.L().Info("received key", zap.String("key", key))
	}()

	// Open a IVF file and start reading using our IVFReader
	file, err := os.Open("../../test/input.ivf")
	if err != nil {
		panic(err)
	}

	ivf, header, err := ivfreader.NewWith(file)
	if err != nil {
		panic(err)
	}

	delta := time.Duration((float32(header.TimebaseNumerator) / float32(header.TimebaseDenominator)) * 1000)
	ticker := time.NewTicker(time.Millisecond * delta)
	for range ticker.C {
		frame, _, err := ivf.ParseNextFrame()
		if err == io.EOF {
			return
		}

		if err != nil {
			panic(err)
		}

		if err = tl.WriteSample(media.Sample{Data: frame, Duration: delta}); err != nil {
			panic(err)
		}
	}
}
