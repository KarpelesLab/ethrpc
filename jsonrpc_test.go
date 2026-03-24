package ethrpc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRequest(t *testing.T) {
	req := NewRequest("eth_blockNumber")
	if req.JsonRpc != "2.0" {
		t.Errorf("JsonRpc = %q, want %q", req.JsonRpc, "2.0")
	}
	if req.Method != "eth_blockNumber" {
		t.Errorf("Method = %q, want %q", req.Method, "eth_blockNumber")
	}
	// Params should be empty slice, not nil
	params, ok := req.Params.([]any)
	if !ok {
		t.Fatalf("Params type = %T, want []any", req.Params)
	}
	if len(params) != 0 {
		t.Errorf("Params length = %d, want 0", len(params))
	}
	// Id should be non-zero
	id, ok := req.Id.(uint64)
	if !ok || id == 0 {
		t.Errorf("Id = %v, want non-zero uint64", req.Id)
	}
}

func TestNewRequestWithParams(t *testing.T) {
	req := NewRequest("eth_getBalance", "0xdead", "latest")
	params, ok := req.Params.([]any)
	if !ok {
		t.Fatalf("Params type = %T, want []any", req.Params)
	}
	if len(params) != 2 {
		t.Fatalf("Params length = %d, want 2", len(params))
	}
	if params[0] != "0xdead" || params[1] != "latest" {
		t.Errorf("Params = %v, want [0xdead latest]", params)
	}
}

func TestNewRequestMap(t *testing.T) {
	req := NewRequestMap("eth_call", map[string]any{"to": "0xdead"})
	if req.Method != "eth_call" {
		t.Errorf("Method = %q, want %q", req.Method, "eth_call")
	}
	params, ok := req.Params.(map[string]any)
	if !ok {
		t.Fatalf("Params type = %T, want map[string]any", req.Params)
	}
	if params["to"] != "0xdead" {
		t.Errorf("Params[to] = %v, want 0xdead", params["to"])
	}
}

func TestNewRequestMapNil(t *testing.T) {
	req := NewRequestMap("eth_call", nil)
	params, ok := req.Params.(map[string]any)
	if !ok {
		t.Fatalf("Params type = %T, want map[string]any", req.Params)
	}
	if len(params) != 0 {
		t.Errorf("Params length = %d, want 0", len(params))
	}
}

func TestRequestHTTPRequest(t *testing.T) {
	req := NewRequest("eth_blockNumber")
	hreq, err := req.HTTPRequest(context.Background(), "http://localhost:8545")
	if err != nil {
		t.Fatal(err)
	}
	if hreq.Method != http.MethodPost {
		t.Errorf("HTTP method = %q, want POST", hreq.Method)
	}
	if ct := hreq.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	// Verify GetBody works
	body, err := hreq.GetBody()
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	var decoded Request
	if err := json.NewDecoder(body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Method != "eth_blockNumber" {
		t.Errorf("decoded method = %q, want eth_blockNumber", decoded.Method)
	}
}

func TestErrorObjectError(t *testing.T) {
	e := &ErrorObject{Code: -32601, Message: "Method not found"}
	got := e.Error()
	want := "jsonrpc error -32601: Method not found"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestMakeError(t *testing.T) {
	req := NewRequest("eth_test")

	t.Run("generic error", func(t *testing.T) {
		resp := req.makeError(errors.New("something broke"))
		if resp.Error == nil {
			t.Fatal("expected error in response")
		}
		if resp.Error.Code != -32603 {
			t.Errorf("error code = %d, want -32603", resp.Error.Code)
		}
		if resp.Error.Message != "something broke" {
			t.Errorf("error message = %q, want %q", resp.Error.Message, "something broke")
		}
	})

	t.Run("ErrorObject passthrough", func(t *testing.T) {
		original := &ErrorObject{Code: -32601, Message: "Method not found"}
		resp := req.makeError(original)
		if resp.Error.Code != -32601 {
			t.Errorf("error code = %d, want -32601", resp.Error.Code)
		}
	})
}

func TestRPCSendCtx(t *testing.T) {
	// Set up a test HTTP server that returns a valid JSON-RPC response
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := Response{
			JsonRpc: "2.0",
			Result:  json.RawMessage(`"0x1b4"`),
			Id:      req.Id,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	rpc := New(srv.URL)

	t.Run("basic call", func(t *testing.T) {
		result, err := rpc.Do("eth_blockNumber")
		if err != nil {
			t.Fatal(err)
		}
		val, err := ReadUint64(result, nil)
		if err != nil {
			t.Fatal(err)
		}
		if val != 436 {
			t.Errorf("got %d, want 436", val)
		}
	})

	t.Run("To helper", func(t *testing.T) {
		var s string
		if err := rpc.To(&s, "eth_blockNumber"); err != nil {
			t.Fatal(err)
		}
		if s != "0x1b4" {
			t.Errorf("got %q, want %q", s, "0x1b4")
		}
	})
}

func TestRPCSendCtxError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		json.NewDecoder(r.Body).Decode(&req)
		resp := map[string]any{
			"jsonrpc": "2.0",
			"error":   map[string]any{"code": -32601, "message": "Method not found"},
			"id":      req.Id,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	rpc := New(srv.URL)
	_, err := rpc.Do("nonexistent_method")
	if err == nil {
		t.Fatal("expected error")
	}
	var eo *ErrorObject
	if !errors.As(err, &eo) {
		t.Fatalf("expected ErrorObject in chain, got %T: %v", err, err)
	}
	if eo.Code != -32601 {
		t.Errorf("error code = %d, want -32601", eo.Code)
	}
}

func TestRPCOverride(t *testing.T) {
	rpc := New("")
	rpc.Override("test_method", func(ctx context.Context) (string, error) {
		return "overridden", nil
	})

	result, err := rpc.Do("test_method")
	if err != nil {
		t.Fatal(err)
	}
	var s string
	if err := json.Unmarshal(result, &s); err != nil {
		t.Fatal(err)
	}
	if s != "overridden" {
		t.Errorf("got %q, want %q", s, "overridden")
	}
}

func TestRPCNoHost(t *testing.T) {
	rpc := New("")
	_, err := rpc.Do("eth_blockNumber")
	if err == nil {
		t.Fatal("expected error for empty host with no override")
	}
}

func TestRPCBasicAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "myuser" || pass != "mypass" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var req Request
		json.NewDecoder(r.Body).Decode(&req)
		json.NewEncoder(w).Encode(Response{
			JsonRpc: "2.0",
			Result:  json.RawMessage(`"ok"`),
			Id:      req.Id,
		})
	}))
	defer srv.Close()

	rpc := New(srv.URL)
	rpc.SetBasicAuth("myuser", "mypass")

	var s string
	if err := rpc.To(&s, "test"); err != nil {
		t.Fatal(err)
	}
	if s != "ok" {
		t.Errorf("got %q, want %q", s, "ok")
	}
}
