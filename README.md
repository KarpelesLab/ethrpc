[![GoDoc](https://godoc.org/github.com/KarpelesLab/ethrpc?status.svg)](https://godoc.org/github.com/KarpelesLab/ethrpc)

# ethrpc

Simple go lib to make RPC calls to Ethereum-like nodes easy

## Example use

```go
    target := ethrpc.New("https://cloudflare-eth.com")
    currentBlockNo, err := ethrpc.ReadUint64(target.Do("eth_blockNumber"))
```

## TODO

* Support websocket
