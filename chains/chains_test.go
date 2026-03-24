package chains

import "testing"

func TestGetEthereumMainnet(t *testing.T) {
	ci := Get(1)
	if ci == nil {
		t.Fatal("Get(1) returned nil, expected Ethereum Mainnet")
	}
	if ci.Name != "Ethereum Mainnet" {
		t.Errorf("Name = %q, want %q", ci.Name, "Ethereum Mainnet")
	}
	if ci.ChainId != 1 {
		t.Errorf("ChainId = %d, want 1", ci.ChainId)
	}
	if ci.NativeCurrency == nil || ci.NativeCurrency.Symbol != "ETH" {
		t.Error("expected native currency ETH")
	}
}

func TestGetCached(t *testing.T) {
	// First call populates cache
	ci1 := Get(1)
	// Second call should return cached value (same pointer)
	ci2 := Get(1)
	if ci1 != ci2 {
		t.Error("expected cached result to return same pointer")
	}
}

func TestGetUnknownChain(t *testing.T) {
	ci := Get(0)
	if ci != nil {
		t.Errorf("Get(0) = %v, want nil", ci)
	}
}

func TestHasFeature(t *testing.T) {
	ci := Get(1) // Ethereum Mainnet has EIP155 and EIP1559
	if ci == nil {
		t.Fatal("Get(1) returned nil")
	}
	if !ci.HasFeature("EIP155") {
		t.Error("expected Ethereum Mainnet to have EIP155")
	}
	if !ci.HasFeature("EIP1559") {
		t.Error("expected Ethereum Mainnet to have EIP1559")
	}
	if ci.HasFeature("nonexistent") {
		t.Error("expected HasFeature(nonexistent) to return false")
	}
}

func TestTransactionUrl(t *testing.T) {
	ci := Get(1)
	if ci == nil {
		t.Fatal("Get(1) returned nil")
	}
	url := ci.TransactionUrl("0xabc123")
	want := "https://etherscan.io/tx/0xabc123"
	if url != want {
		t.Errorf("TransactionUrl = %q, want %q", url, want)
	}
}

func TestTransactionUrlNoExplorer(t *testing.T) {
	ci := &ChainInfo{Name: "Test"}
	if url := ci.TransactionUrl("0xabc"); url != "" {
		t.Errorf("expected empty string, got %q", url)
	}
}

func TestExplorerURL(t *testing.T) {
	ci := Get(1)
	if ci == nil {
		t.Fatal("Get(1) returned nil")
	}
	url := ci.ExplorerURL()
	if url != "https://etherscan.io" {
		t.Errorf("ExplorerURL = %q, want %q", url, "https://etherscan.io")
	}
}

func TestExplorerURLEmpty(t *testing.T) {
	ci := &ChainInfo{Name: "Test"}
	if url := ci.ExplorerURL(); url != "" {
		t.Errorf("expected empty string, got %q", url)
	}
}
