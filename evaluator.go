package ethrpc

import (
	"context"
	"encoding/json"
	"time"
)

// RPCList is a list of [RPC] endpoints that implements [Handler].
type RPCList []*RPC

// DoCtx performs a JSON-RPC call using the first server in the list.
func (r RPCList) DoCtx(ctx context.Context, method string, args ...any) (json.RawMessage, error) {
	if len(r) == 0 {
		return nil, ErrNoAvailableServer
	}
	// TODO might want to be able to fallback to the next server in list or other fancy things...
	return r[0].DoCtx(ctx, method, args...)
}

// Evaluate will call the various servers in the list and return a list of servers that work (if any)
//
// This will send a eth_blockNumber request to all the servers and measure the response time
func Evaluate(ctx context.Context, servers ...string) (Handler, error) {
	if len(servers) == 0 {
		return nil, ErrNoAvailableServer
	}
	if len(servers) == 1 {
		// only 1 server, return it
		return New(servers[0]), nil
	}

	// make sure to cancel any pending request if we end
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	count := len(servers)
	rech := make(chan *RPC, count+1)
	errch := make(chan error, count+1)

	for _, s := range servers {
		go func(s string) {
			r := New(s)
			start := time.Now()
			res, err := ReadUint64(r.DoCtx(ctx, "eth_blockNumber"))
			if err != nil {
				errch <- err
				return
			}
			r.lag = time.Since(start)
			r.block = res
			rech <- r
		}(s)
	}

	var (
		timer *time.Timer
		c     <-chan time.Time
		res   RPCList
	)

	for {
		select {
		case <-c:
			// timeout on selection
			return res, nil
		case <-ctx.Done():
			return res, ctx.Err()
		case v := <-rech:
			res = append(res, v)
			count -= 1
			if count == 0 {
				// end of the list but we got at least 1
				return res, nil
			}
			// setup timer to end 200ms after the first successful response, so we don't spend too long waiting
			if timer == nil {
				timer = time.NewTimer(200 * time.Millisecond)
				defer timer.Stop()
				c = timer.C
			}
		case e := <-errch:
			count -= 1
			if count == 0 {
				// nothing more
				if len(res) > 0 {
					return res, nil
				} else {
					// nothing to return either, so return the last error we got
					return nil, e
				}
			}
		}
	}
}
