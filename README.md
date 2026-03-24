[![GoDoc](https://godoc.org/github.com/KarpelesLab/ethrpc?status.svg)](https://godoc.org/github.com/KarpelesLab/ethrpc)

# ethrpc

A lightweight Go library for making JSON-RPC calls to Ethereum-compatible nodes.

## Install

```bash
go get github.com/KarpelesLab/ethrpc
```

## Quick start

```go
rpc := ethrpc.New("https://cloudflare-eth.com")
blockNo, err := ethrpc.ReadUint64(rpc.Do("eth_blockNumber"))
```

## Features

### Positional and named arguments

```go
// Positional arguments
balance, err := ethrpc.ReadBigInt(rpc.Do("eth_getBalance", "0xAddress", "latest"))

// Named arguments
result, err := rpc.DoNamed("eth_call", map[string]any{
    "to":   "0xContract",
    "data": "0xCalldata",
})
```

### Decode helpers

Response decoders can be chained directly with `Do` calls:

```go
blockNo, err := ethrpc.ReadUint64(rpc.Do("eth_blockNumber"))
balance, err := ethrpc.ReadBigInt(rpc.Do("eth_getBalance", addr, "latest"))
hash, err := ethrpc.ReadString(rpc.Do("eth_sendRawTransaction", signedTx))

// Decode into a struct
var block MyBlockType
err := ethrpc.ReadTo(&block)(rpc.Do("eth_getBlockByNumber", "0x1b4", true))

// Generic decoder
block, err := ethrpc.ReadAs[MyBlockType](rpc.Do("eth_getBlockByNumber", "0x1b4", true))
```

### Unmarshal into a target

```go
var peers []any
err := rpc.To(&peers, "net_peerCount")
```

### Context support

All methods have context-aware variants:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

blockNo, err := ethrpc.ReadUint64(rpc.DoCtx(ctx, "eth_blockNumber"))
```

### Basic authentication

```go
rpc := ethrpc.New("https://my-node.example.com")
rpc.SetBasicAuth("user", "password")
```

### Method overrides

Intercept RPC methods locally without hitting the remote node:

```go
rpc.Override("eth_chainId", func(ctx context.Context) (string, error) {
    return "0x1", nil
})
```

### Server evaluation

Select the fastest endpoint from a list by racing `eth_blockNumber` calls:

```go
handler, err := ethrpc.Evaluate(ctx,
    "https://node1.example.com",
    "https://node2.example.com",
    "https://node3.example.com",
)
// handler implements ethrpc.Handler with the best responding servers
```

### HTTP response forwarding

Proxy JSON-RPC responses directly to an `http.ResponseWriter`:

```go
rpc.Forward(ctx, w, req, &ethrpc.ForwardOptions{
    Pretty: true,
    Cache:  30 * time.Second,
})
```

### Chain metadata

The `chains` subpackage provides static metadata for known EVM-compatible chains:

```go
import "github.com/KarpelesLab/ethrpc/chains"

eth := chains.Get(1) // Ethereum Mainnet
fmt.Println(eth.Name)                          // "Ethereum Mainnet"
fmt.Println(eth.NativeCurrency.Symbol)         // "ETH"
fmt.Println(eth.HasFeature("EIP1559"))         // true
fmt.Println(eth.TransactionUrl("0xabc..."))    // "https://etherscan.io/tx/0xabc..."
fmt.Println(eth.ExplorerURL())                 // "https://etherscan.io"
```
