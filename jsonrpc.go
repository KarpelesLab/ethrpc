package ethrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
)

// Request represents a JSON-RPC 2.0 request object.
type Request struct {
	JsonRpc string `json:"jsonrpc"` // 2.0
	Method  string `json:"method"`
	Params  any    `json:"params"`
	Id      any    `json:"id"`
}

// NewRequest makes a new [Request] fit to use with methods like Send.
func NewRequest(method string, params ...any) *Request {
	if params == nil {
		// make sure it is not nil so json will encode it as "[]" and not "null"
		params = []any{}
	}
	req := &Request{
		JsonRpc: "2.0",
		Method:  method,
		Params:  params,
		Id:      atomic.AddUint64(&rpcId, 1),
	}
	return req
}

// NewRequestMap makes a new [Request] fit to use with methods like Send, using named parameters.
func NewRequestMap(method string, params map[string]any) *Request {
	if params == nil {
		// make sure it is not nil
		params = make(map[string]any)
	}
	req := &Request{
		JsonRpc: "2.0",
		Method:  method,
		Params:  params,
		Id:      atomic.AddUint64(&rpcId, 1),
	}
	return req
}

// HTTPRequest returns a [http.Request] for the given json-rpc request.
func (req *Request) HTTPRequest(ctx context.Context, host string) (*http.Request, error) {
	reqEnc, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s request: %w", req.Method, err)
	}

	hreq, err := http.NewRequestWithContext(ctx, "POST", host, bytes.NewReader(reqEnc))
	if err != nil {
		return nil, fmt.Errorf("failed to generate HTTP request for %s: %w", req.Method, err)
	}
	hreq.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(reqEnc)), nil }
	hreq.Header.Set("Content-Type", "application/json")

	return hreq, nil
}

func (r *Request) makeError(e error) *ResponseIntf {
	res := &ResponseIntf{
		JsonRpc: "2.0",
		Id:      r.Id,
	}

	if !errors.As(e, &res.Error) {
		res.Error = &ErrorObject{
			Message: e.Error(),
			Code:    -32603,
		}
	}
	return res
}

// Response represents a JSON-RPC 2.0 response with a raw JSON result.
type Response struct {
	JsonRpc string          `json:"jsonrpc"` // 2.0
	Result  json.RawMessage `json:"result"`
	Error   *ErrorObject    `json:"error,omitempty"`
	Id      any             `json:"id"`
}

// ResponseIntf is a JSON-RPC 2.0 response where Result is an arbitrary type
// rather than [json.RawMessage], allowing direct encoding of Go values.
type ResponseIntf struct {
	JsonRpc string       `json:"jsonrpc"` // 2.0
	Result  any          `json:"result"`
	Error   *ErrorObject `json:"error,omitempty"`
	Id      any          `json:"id"`
}

// ErrorObject represents a JSON-RPC 2.0 error returned by the server.
type ErrorObject struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *ErrorObject) Error() string {
	return fmt.Sprintf("jsonrpc error %d: %s", e.Code, e.Message)
}
