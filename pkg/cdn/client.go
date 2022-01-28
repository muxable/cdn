package cdn

import (
	"context"
	"errors"

	"github.com/muxable/cdn/api"
	"github.com/muxable/cdn/internal/signal"
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

func (c *Client) Publish(ctx context.Context, tl webrtc.TrackLocal) (string, error) {
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		return "", err
	}

	signaller := signal.Negotiate(peerConnection)

	publish, err := c.grpcClient.Publish(ctx)
	if err != nil {
		return "", err
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

	output := make(chan string)
	defer close(output)

	peerConnection.OnSignalingStateChange(func(ss webrtc.SignalingState) {
		zap.L().Debug("signaling state change", zap.String("state", ss.String()))
	})

	peerConnection.OnICEConnectionStateChange(func(c webrtc.ICEConnectionState) {
		zap.L().Debug("ice connection state change", zap.String("state", c.String()))
	})

	go func() {
		for {
			in, err := publish.Recv()
			if err != nil {
				return
			}

			switch operation := in.Operation.(type) {
			case *api.PublishResponse_Track:
				output <- operation.Track.Key
			case *api.PublishResponse_Signal:
				if err := signaller.WriteSignal(operation.Signal); err != nil {
					zap.L().Error("failed to write signal", zap.Error(err))
					return
				}
			}
		}
	}()

	rtpSender, err := peerConnection.AddTrack(tl)
	if err != nil {
		return "", err
	}

	go func() {
		buf := make([]byte, 1500)
		for {
			_, _, err := rtpSender.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	tr := <-output
	if err := publish.CloseSend(); err != nil {
		return "", err
	}
	return tr, nil
}

type SubscriberConfiguration func(*api.SubscribeRequest_Subscription)

type Subscription struct {
	*webrtc.TrackRemote

	Trace []string
}

func (c *Client) Subscribe(ctx context.Context, key string, options ...SubscriberConfiguration) (*Subscription, error) {
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		return nil, err
	}

	signaller := signal.Negotiate(peerConnection)

	subscribe, err := c.grpcClient.Subscribe(ctx)
	if err != nil {
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

	trCh := make(chan *webrtc.TrackRemote, 1)
	traceCh := make(chan []string, 1)

	peerConnection.OnTrack(func(tr *webrtc.TrackRemote, r *webrtc.RTPReceiver) {
		go func() {
			buf := make([]byte, 1500)
			for {
				if _, _, err := r.Read(buf); err != nil {
					return
				}
			}
		}()

		trCh <- tr
	})

	subscription := &api.SubscribeRequest_Subscription{
		Key: key,
		// TODO: include other fields.
	}

	for _, option := range options {
		option(subscription)
	}

	request := &api.SubscribeRequest{
		Operation: &api.SubscribeRequest_Subscription_{
			Subscription: subscription,
		},
	}

	if err := subscribe.Send(request); err != nil {
		return nil, err
	}

	go func() {
		defer close(trCh)
		defer close(traceCh)
		for {
			in, err := subscribe.Recv()
			if err != nil {
				zap.L().Error("failed to receive", zap.Error(err))
				return
			}

			switch operation := in.Operation.(type) {
			case *api.SubscribeResponse_Track:
				// assume this is the relevant track because we only added one per peer connection.
				traceCh <- operation.Track.Trace
			case *api.SubscribeResponse_Signal:
				if err := signaller.WriteSignal(operation.Signal); err != nil {
					return
				}
			}
		}
	}()

	var tr *webrtc.TrackRemote
	var trace []string
	var ok bool
	for tr == nil || trace == nil {
		select {
		case tr, ok = <-trCh:
			if !ok {
				return nil, errors.New("track not found")
			}
		case trace, ok = <-traceCh:
			if !ok {
				return nil, errors.New("track not found")
			}
		}
	}
	return &Subscription{TrackRemote: tr, Trace: trace}, nil
}
