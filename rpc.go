package ethrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/KarpelesLab/typutil"
)

// Note: see https://eth.wiki/json-rpc/API for APIs
// Note: see https://ethereum.org/en/developers/docs/apis/json-rpc/ for APIs

var rpcId uint64

// RPC represents a connection to an Ethereum JSON-RPC endpoint.
// It supports HTTP transport with optional basic authentication and method overrides.
type RPC struct {
	host       string
	lag        time.Duration // how long it takes for this endpoint to respond to eth_blockNumber
	block      uint64        // latest block number
	HTTPClient *http.Client
	// for RPC auth
	username string
	password string
	override map[string]*typutil.Callable
}

// New returns a new instance of RPC to perform requests to the given RPC endpoint
func New(h string) *RPC {
	return &RPC{host: h, HTTPClient: http.DefaultClient, override: make(map[string]*typutil.Callable)}
}

// Override allows redirecting calls to a RPC method to a standard go function
func (r *RPC) Override(method string, fnc any) {
	r.override[method] = typutil.Func(fnc)
}

// SetHost sets the host requests are sent to for subsequent RPC requests
func (r *RPC) SetHost(host string) {
	r.host = host
}

// GetHost returns the host as configured
func (r *RPC) GetHost() string {
	return r.host
}

// SetBasicAuth sets basic auth params for all subsequent RPC requests
func (r *RPC) SetBasicAuth(username, password string) {
	r.username = username
	r.password = password
}

// Do performs a RPC request
func (r *RPC) Do(method string, args ...any) (json.RawMessage, error) {
	return r.DoCtx(context.Background(), method, args...)
}

// DoNamed performs a RPC request using named arguments
func (r *RPC) DoNamed(method string, args map[string]any) (json.RawMessage, error) {
	return r.DoNamedCtx(context.Background(), method, args)
}

// Send sends a raw request for processing
func (r *RPC) Send(req *Request) (json.RawMessage, error) {
	return r.SendCtx(context.Background(), req)
}

// DoCtx performs a RPC request, taking an optional context that can be cancelled to stop the request
func (r *RPC) DoCtx(ctx context.Context, method string, args ...any) (json.RawMessage, error) {
	return r.SendCtx(ctx, NewRequest(method, args...))
}

// DoNamedCtx performs a RPC request using named arguments, taking an optional context that can be cancelled to stop the request
func (r *RPC) DoNamedCtx(ctx context.Context, method string, args map[string]any) (json.RawMessage, error) {
	return r.SendCtx(ctx, NewRequest(method, args))
}

// SendCtx sends a raw [Request] to the RPC endpoint using the provided context.
func (r *RPC) SendCtx(ctx context.Context, req *Request) (json.RawMessage, error) {
	// JSON RPC over http is simple
	//log.Printf("[RPC] → %s %v", method, args)

	if f, ok := r.override[req.Method]; ok {
		if req.Params == nil {
			req.Params = []any{}
		}
		if params, ok := req.Params.([]any); ok {
			res, err := f.CallArg(ctx, params...)
			if err != nil {
				return nil, err
			}
			return json.Marshal(res)
		}
		return nil, errors.New("function requires positional arguments instead of named arguments")
	}

	if r.host == "" {
		// special case, we only process override, anything else will be not found
		return nil, fs.ErrNotExist
	}

	hreq, err := req.HTTPRequest(ctx, r.host)
	if err != nil {
		return nil, fmt.Errorf("failed to generate HTTP request for %s: %w", req.Method, err)
	}

	if r.username != "" || r.password != "" {
		hreq.SetBasicAuth(r.username, r.password)
	}

	// post it
	resp, err := r.HTTPClient.Do(hreq)
	if err != nil {
		return nil, fmt.Errorf("error while performing %s: %w", req.Method, err)
	}
	defer resp.Body.Close()

	// decode response
	reader := json.NewDecoder(resp.Body)
	var res *Response
	err = reader.Decode(&res)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response to %s: %w", req.Method, err)
	}
	if res.Error != nil {
		//log.Printf("[RPC] ← Error: %s", res.Error.Error())
		return nil, fmt.Errorf("RPC error during %s: %w", req.Method, res.Error)
	}

	//log.Printf("[RPC] ← %s", res.Result)

	return res.Result, nil
}

// To performs the request and puts the result into target
func (r *RPC) To(target any, method string, args ...any) error {
	v, err := r.Do(method, args...)
	if err != nil {
		return err
	}
	return json.Unmarshal(v, target)
}

// ForwardOptions configures how [RPC.Forward] writes the response.
type ForwardOptions struct {
	Pretty bool
	Cache  time.Duration
}

// Forward will write the RPC response to the given [http.ResponseWriter].
func (r *RPC) Forward(ctx context.Context, rw http.ResponseWriter, req *Request, opts *ForwardOptions) {
	if f, ok := r.override[req.Method]; ok {
		// do not forward but run locally
		rw.Header().Set("Content-Type", "application/json")
		rw.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if opts != nil && opts.Cache > 0 {
			rw.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", opts.Cache/time.Second))
			rw.Header().Set("Expires", time.Now().Add(opts.Cache).Format(time.RFC1123))
		}
		enc := json.NewEncoder(rw)
		if opts != nil && opts.Pretty {
			enc.SetIndent("", "    ")
		}
		if req.Params == nil {
			req.Params = []any{}
		}
		if params, ok := req.Params.([]any); ok {
			res, err := f.CallArg(ctx, params...)
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				enc.Encode(req.makeError(err))
				return
			}
			enc.Encode(&ResponseIntf{JsonRpc: "2.0", Result: res, Id: req.Id})
		} else {
			rw.WriteHeader(http.StatusBadRequest)
			enc.Encode(req.makeError(errors.New("function only supports positional arguments")))
		}
		return
	}

	if r.host == "" {
		// not found
		http.Error(rw, "404 page not found", http.StatusNotFound)
		return
	}

	// json rpc request forwarded to a response writer
	// First, let's do the request
	hreq, err := req.HTTPRequest(ctx, r.host)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.username != "" || r.password != "" {
		hreq.SetBasicAuth(r.username, r.password)
	}

	// post it
	resp, err := r.HTTPClient.Do(hreq)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		rw.Header()[k] = v
	}
	rw.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

	if opts != nil && opts.Cache > 0 {
		rw.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", opts.Cache/time.Second))
		rw.Header().Set("Expires", time.Now().Add(opts.Cache).Format(time.RFC1123))
	}

	if opts != nil && opts.Pretty {
		// remove Content-Length from headers
		rw.Header().Del("Content-Length")

		// send the rest
		rw.WriteHeader(resp.StatusCode)
		// special case: we need to format the response json
		buf, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("[http] failed to read response: %s", err)
			return
		}
		outbuf := &bytes.Buffer{}
		err = json.Indent(outbuf, buf, "", "  ")
		if err != nil {
			rw.Write(buf)
		} else {
			io.Copy(rw, outbuf)
		}
		return
	}

	rw.WriteHeader(resp.StatusCode)
	io.Copy(rw, resp.Body)
}
