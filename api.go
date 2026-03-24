package ethrpc

import (
	"context"
	"encoding/json"
)

// Handler is the interface for any backend capable of executing JSON-RPC calls.
type Handler interface {
	DoCtx(ctx context.Context, method string, args ...any) (json.RawMessage, error)
}

// Api wraps a [Handler] and provides convenience methods for common Ethereum RPC calls.
type Api struct {
	Handler
}

// Do performs a JSON-RPC call using a background context.
func (a *Api) Do(method string, args ...any) (json.RawMessage, error) {
	return a.Handler.DoCtx(context.Background(), method, args...)
}

// To performs a JSON-RPC call and unmarshals the result into target.
func (a *Api) To(target any, method string, args ...any) error {
	return a.ToCtx(context.Background(), target, method, args...)
}

// ToCtx performs a JSON-RPC call with context and unmarshals the result into target.
func (a *Api) ToCtx(ctx context.Context, target any, method string, args ...any) error {
	v, err := a.DoCtx(ctx, method, args...)
	if err != nil {
		return err
	}
	return json.Unmarshal(v, target)
}

// BlockNumber returns the current block number from the connected node.
func (a *Api) BlockNumber(ctx context.Context) (uint64, error) {
	return ReadUint64(a.Handler.DoCtx(ctx, "eth_blockNumber"))
}

// ChainId returns the chain ID of the connected network.
func (a *Api) ChainId(ctx context.Context) (uint64, error) {
	return ReadUint64(a.Handler.DoCtx(ctx, "eth_chainId"))
}
