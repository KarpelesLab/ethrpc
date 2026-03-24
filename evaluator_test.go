package ethrpc

import (
	"context"
	"encoding/json"
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
