// Package chains provides static metadata for known EVM-compatible chains.
package chains

import "fmt"

// ChainFeature represents a feature supported by a chain (e.g. EIP155, EIP1559).
type ChainFeature struct {
	Name string `json:"name"`
}

// ChainCurrency describes a chain's native currency.
type ChainCurrency struct {
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}

// ChainENS holds ENS registry information for a chain.
type ChainENS struct {
	Registry string `json:"registry"`
}

// ChainExplorer describes a block explorer for a chain.
type ChainExplorer struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Standard string `json:"standard"` // EIP3091
}

// ChainInfo holds metadata for an EVM-compatible chain.
type ChainInfo struct {
	Name           string           `json:"name"`
	Chain          string           `json:"chain"`
	Icon           string           `json:"icon"`
	RPC            []string         `json:"rpc"`
	Features       []*ChainFeature  `json:"features"`
	Faucets        []string         `json:"faucets"`
	NativeCurrency *ChainCurrency   `json:"nativeCurrency"`
	InfoURL        string           `json:"infoURL"`
	ShortName      string           `json:"shortName"`
	ChainId        uint64           `json:"chainId"`
	NetworkId      uint64           `json:"networkId"`
	Slip44         int              `json:"slip44,omitempty"`
	ENS            *ChainENS        `json:"ens"`
	Explorers      []*ChainExplorer `json:"explorers"`
}

// HasFeature reports whether the chain supports the named feature.
func (ci *ChainInfo) HasFeature(feat string) bool {
	for _, s := range ci.Features {
		if s.Name == feat {
			return true
		}
	}
	return false
}

// TransactionUrl returns a URL to view the given transaction hash on the chain's first explorer.
// Returns an empty string if no explorer is configured.
func (ci *ChainInfo) TransactionUrl(txHash string) string {
	if len(ci.Explorers) == 0 {
		return ""
	}
	return fmt.Sprintf("%s/tx/%s", ci.Explorers[0].URL, txHash)
}

// ExplorerURL returns the URL of the chain's first block explorer, or an empty string if none.
func (ci *ChainInfo) ExplorerURL() string {
	if len(ci.Explorers) > 0 {
		return ci.Explorers[0].URL
	}
	return ""
}
