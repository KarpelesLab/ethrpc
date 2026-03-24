package chains

import (
	"encoding/json"
	"sync"
)

var (
	chainCache = make(map[uint64]*ChainInfo)
	lk         sync.RWMutex
)

// Get returns the [ChainInfo] for the given chain ID, or nil if unknown.
// Results are cached after the first lookup.
func Get(id uint64) *ChainInfo {
	lk.RLock()
	if ci, ok := chainCache[id]; ok {
		lk.RUnlock()
		return ci
	}
	lk.RUnlock()

	buf, ok := chainJSON[id]
	if !ok {
		return nil
	}
	var res *ChainInfo
	err := json.Unmarshal([]byte(buf), &res)
	if err != nil {
		return nil
	}

	lk.Lock()
	chainCache[id] = res
	lk.Unlock()

	return res
}
