package ethrpc

import "errors"

// ErrNoAvailableServer is returned by [Evaluate] when no servers are provided or reachable.
var ErrNoAvailableServer = errors.New("no available server")
