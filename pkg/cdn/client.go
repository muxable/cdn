package cdn

import (
	"context"

	"github.com/muxable/cdn/api"
	"github.com/muxable/signal/pkg/signal"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Client struct {
	grpcClient api.CDNClient
}

func NewClient(conn *grpc.ClientConn) (*Client, error) {
	return &Client{grpcClient: api.NewCDNClient(conn)}, nil
}

func (c *Client) Publish() (*webrtc.PeerConnection, error) {
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		return nil, err
	}

	signaller := signal.NewSignaller(peerConnection)

	peerConnection.OnNegotiationNeeded(signaller.Renegotiate)

	ctx, cancel := context.WithCancel(context.Background())

	publish, err := c.grpcClient.Publish(ctx)
	if err != nil {
		cancel()
		return nil, err
	}

	go func() {
		for {
			signal, err := signaller.ReadSignal()
			if err != nil {
				zap.L().Error("failed to read signal", zap.Error(err))
				return
			}
			if err := publish.Send(&api.PublishRequest{Signal: signal}); err != nil {
				zap.L().Error("failed to send signal", zap.Error(err))
				return
			}
		}
	}()

	peerConnection.OnSignalingStateChange(func(ss webrtc.SignalingState) {
		zap.L().Debug("signaling state change", zap.String("state", ss.String()))
	})

	peerConnection.OnICEConnectionStateChange(func(c webrtc.ICEConnectionState) {
		zap.L().Debug("ice connection state change", zap.String("state", c.String()))
		if c == webrtc.ICEConnectionStateClosed {
			cancel()
		}
	})

	go func() {
		for {
			in, err := publish.Recv()
			if err != nil {
				return
			}

			if err := signaller.WriteSignal(in.Signal); err != nil {
				zap.L().Error("failed to write signal", zap.Error(err))
				return
			}
		}
	}()

	return peerConnection, nil
}

type SubscriberConfiguration func(*api.SubscribeRequest_Subscription)

func (c *Client) Subscribe(key string, options ...SubscriberConfiguration) (*webrtc.PeerConnection, error) {
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		return nil, err
	}

	signaller := signal.NewSignaller(peerConnection)

	peerConnection.OnNegotiationNeeded(signaller.Renegotiate)

	ctx, cancel := context.WithCancel(context.Background())

	subscribe, err := c.grpcClient.Subscribe(ctx)
	if err != nil {
		cancel()
		return nil, err
	}

	go func() {
		for {
			signal, err := signaller.ReadSignal()
			if err != nil {
				zap.L().Error("failed to read signal", zap.Error(err))
				return
			}
			if err := subscribe.Send(&api.SubscribeRequest{
				Operation: &api.SubscribeRequest_Signal{Signal: signal},
			}); err != nil {
				zap.L().Error("failed to send signal", zap.Error(err))
				return
			}
		}
	}()

	subscription := &api.SubscribeRequest_Subscription{
		StreamId: key,
	}

	for _, option := range options {
		option(subscription)
	}

	request := &api.SubscribeRequest{
		Operation: &api.SubscribeRequest_Subscription_{
			Subscription: subscription,
		},
	}

	peerConnection.OnICEConnectionStateChange(func(c webrtc.ICEConnectionState) {
		zap.L().Debug("ice connection state change", zap.String("state", c.String()))
		if c == webrtc.ICEConnectionStateClosed {
			cancel()
		}
	})

	if err := subscribe.Send(request); err != nil {
		return nil, err
	}

	go func() {
		for {
			in, err := subscribe.Recv()
			if err != nil {
				zap.L().Error("failed to receive", zap.Error(err))
				return
			}

			if err := signaller.WriteSignal(in.Signal); err != nil {
				return
			}
		}
	}()

	return peerConnection, nil
}
