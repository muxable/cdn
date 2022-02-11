package getput

import (
	"context"
	"crypto/sha1"
	"errors"
	"math"
	"sync"

	"github.com/anacrolix/dht/v2"
	"github.com/anacrolix/dht/v2/bep44"
	k_nearest_nodes "github.com/anacrolix/dht/v2/k-nearest-nodes"
	"github.com/anacrolix/dht/v2/krpc"
	"github.com/anacrolix/dht/v2/traversal"
	"github.com/anacrolix/torrent/bencode"
	"go.uber.org/zap"
)

type GetResult struct {
	Seq int64
	V   string
}

func Get(ctx context.Context, target bep44.Target, s *dht.Server) (string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	vChan := make(chan GetResult)
	op := traversal.Start(traversal.OperationInput{
		Target: target,
		DoQuery: func(ctx context.Context, addr krpc.NodeAddr) traversal.QueryResult {
			res := s.Get(ctx, dht.NewAddr(addr.UDP()), target, nil, dht.QueryRateLimiting{})
			err := res.ToError()
			if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, dht.TransactionTimeout) {
				zap.L().Error("error querying", zap.Stringer("addr", addr), zap.Error(err))
			}
			if r := res.Reply.R; r != nil {
				rv := r.V
				bv := bencode.MustMarshal(rv)
				if sha1.Sum(r.K[:]) == target && bep44.Verify(r.K[:], nil, *r.Seq, bv, r.Sig[:]) {
					select {
					case vChan <- GetResult{
						Seq: *r.Seq,
						V:   rv.(string),
					}:
					case <-ctx.Done():
					}
				} else if rv != nil {
					zap.L().Error("invalid signature from addr", zap.Stringer("addr", addr))
				}
			}
			return res.TraversalQueryResult(addr)
		},
		NodeFilter: s.TraversalNodeFilter,
	})
	nodes, err := s.TraversalStartingNodes()
	if err != nil {
		return "", err
	}
	op.AddNodes(nodes)
	defer op.Stop()
	best := GetResult{Seq: math.MinInt64}
	for {
		select {
		case <-op.Stalled():
			if best.Seq == math.MinInt64 {
				return "", errors.New("no nodes responded")
			}
			return best.V, nil
		case v := <-vChan:
			if v.Seq >= best.Seq {
				best = v
			}
		case <-ctx.Done():
			return best.V, ctx.Err()
		}
	}
}

func Put(
	ctx context.Context, target krpc.ID, s *dht.Server, put bep44.Put,
) error {
	op := traversal.Start(traversal.OperationInput{
		Alpha:  15,
		Target: target,
		DoQuery: func(ctx context.Context, addr krpc.NodeAddr) traversal.QueryResult {
			res := s.Get(ctx, dht.NewAddr(addr.UDP()), target, nil, dht.QueryRateLimiting{})
			err := res.ToError()
			if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, dht.TransactionTimeout) {
				zap.L().Error("error querying", zap.Stringer("addr", addr), zap.Error(err))
			}
			tqr := res.TraversalQueryResult(addr)
			if tqr.ClosestData == nil {
				tqr.ResponseFrom = nil
			}
			return tqr
		},
		NodeFilter: s.TraversalNodeFilter,
	})
	nodes, err := s.TraversalStartingNodes()
	if err != nil {
		return err
	}
	op.AddNodes(nodes)
	defer op.Stop()
	select {
	case <-op.Stalled():
	case <-ctx.Done():
		return ctx.Err()
	}
	var wg sync.WaitGroup
	success := true
	op.Closest().Range(func(elem k_nearest_nodes.Elem) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token := elem.Data.(string)
			res := s.Put(ctx, dht.NewAddr(elem.Addr.UDP()), put, token, dht.QueryRateLimiting{})
			err := res.ToError()
			if err != nil {
				success = false
			}
		}()
	})
	wg.Wait()
	if !success {
		return errors.New("unable to save track, perhaps it already exists?")
	}
	return nil
}
