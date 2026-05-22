package ethrpc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRPCListDoCtxEmpty(t *testing.T) {
	var list RPCList
	_, err := list.DoCtx(context.Background(), "eth_blockNumber")
	if err != ErrNoAvailableServer {
		t.Errorf("got %v, want ErrNoAvailableServer", err)
	}
}

func TestEvaluateNoServers(t *testing.T) {
	_, err := Evaluate(context.Background())
	if err != ErrNoAvailableServer {
		t.Errorf("got %v, want ErrNoAvailableServer", err)
	}
}

func TestEvaluateSingleServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		json.NewDecoder(r.Body).Decode(&req)
		json.NewEncoder(w).Encode(Response{
			JsonRpc: "2.0",
			Result:  json.RawMessage(`"0x1"`),
			Id:      req.Id,
		})
	}))
	defer srv.Close()

	h, err := Evaluate(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	// Single server should return an *RPC, not an RPCList
	if _, ok := h.(*RPC); !ok {
		t.Errorf("expected *RPC, got %T", h)
	}
}

func TestEvaluateMultipleServers(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		json.NewDecoder(r.Body).Decode(&req)
		json.NewEncoder(w).Encode(Response{
			JsonRpc: "2.0",
			Result:  json.RawMessage(`"0xa"`),
			Id:      req.Id,
		})
	})
	srv1 := httptest.NewServer(handler)
	defer srv1.Close()
	srv2 := httptest.NewServer(handler)
	defer srv2.Close()

	h, err := Evaluate(context.Background(), srv1.URL, srv2.URL)
	if err != nil {
		t.Fatal(err)
	}

	// Should be able to make a call through the returned handler
	result, err := h.DoCtx(context.Background(), "eth_blockNumber")
	if err != nil {
		t.Fatal(err)
	}
	val, err := ReadUint64(result, nil)
	if err != nil {
		t.Fatal(err)
	}
	if val != 10 {
		t.Errorf("got %d, want 10", val)
	}
}

// Regression: RPCList.DoCtx should fail over from a transport error to a working server.
func TestRPCListFailover(t *testing.T) {
	// First server: always 502
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer bad.Close()
	// Second server: works
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		json.NewDecoder(r.Body).Decode(&req)
		json.NewEncoder(w).Encode(Response{
			JsonRpc: "2.0",
			Result:  json.RawMessage(`"0x2a"`),
			Id:      req.Id,
		})
	}))
	defer good.Close()

	list := RPCList{New(bad.URL), New(good.URL)}
	res, err := list.DoCtx(context.Background(), "eth_blockNumber")
	if err != nil {
		t.Fatalf("expected failover to succeed, got %v", err)
	}
	val, err := ReadUint64(res, nil)
	if err != nil {
		t.Fatal(err)
	}
	if val != 0x2a {
		t.Errorf("got %d, want 42", val)
	}
}

// Regression: a JSON-RPC error from the first server must NOT cause failover
// (the response is valid, just an error).
func TestRPCListNoFailoverOnJSONRPCError(t *testing.T) {
	var goodCalls int
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		json.NewDecoder(r.Body).Decode(&req)
		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"error":   map[string]any{"code": -32601, "message": "Method not found"},
			"id":      req.Id,
		})
	}))
	defer first.Close()
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		goodCalls++
		var req Request
		json.NewDecoder(r.Body).Decode(&req)
		json.NewEncoder(w).Encode(Response{
			JsonRpc: "2.0",
			Result:  json.RawMessage(`"0x1"`),
			Id:      req.Id,
		})
	}))
	defer second.Close()

	list := RPCList{New(first.URL), New(second.URL)}
	_, err := list.DoCtx(context.Background(), "any")
	if err == nil {
		t.Fatal("expected JSON-RPC error")
	}
	var eo *ErrorObject
	if !errors.As(err, &eo) {
		t.Fatalf("expected ErrorObject, got %T: %v", err, err)
	}
	if goodCalls != 0 {
		t.Errorf("second server should not have been called, got %d calls", goodCalls)
	}
}

// Regression: Evaluate with a single failing server must error out instead of
// returning a dead handler.
func TestEvaluateSingleServerFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	_, err := Evaluate(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected evaluate to fail against a broken server")
	}
}
