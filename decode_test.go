package ethrpc

import (
	"encoding/json"
	"errors"
	"math/big"
	"testing"
)

func TestReadUint64(t *testing.T) {
	tests := []struct {
		name    string
		input   json.RawMessage
		err     error
		want    uint64
		wantErr bool
	}{
		{"hex string", json.RawMessage(`"0x1b4"`), nil, 436, false},
		{"decimal string", json.RawMessage(`"100"`), nil, 100, false},
		{"number literal", json.RawMessage(`42`), nil, 42, false},
		{"zero", json.RawMessage(`"0x0"`), nil, 0, false},
		{"passthrough error", nil, errors.New("rpc failed"), 0, true},
		{"invalid string", json.RawMessage(`"notanumber"`), nil, 0, true},
		{"invalid json", json.RawMessage(`{}`), nil, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadUint64(tt.input, tt.err)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ReadUint64() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ReadUint64() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestReadBigInt(t *testing.T) {
	tests := []struct {
		name    string
		input   json.RawMessage
		err     error
		want    *big.Int
		wantErr bool
	}{
		{"hex string", json.RawMessage(`"0x1b4"`), nil, big.NewInt(436), false},
		{"decimal string", json.RawMessage(`"100"`), nil, big.NewInt(100), false},
		{"number literal", json.RawMessage(`42`), nil, big.NewInt(42), false},
		{"large hex", json.RawMessage(`"0xDE0B6B3A7640000"`), nil, new(big.Int).SetUint64(1000000000000000000), false},
		{"passthrough error", nil, errors.New("rpc failed"), nil, true},
		{"invalid string", json.RawMessage(`"notanumber"`), nil, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadBigInt(tt.input, tt.err)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ReadBigInt() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got.Cmp(tt.want) != 0 {
				t.Errorf("ReadBigInt() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestReadString(t *testing.T) {
	tests := []struct {
		name    string
		input   json.RawMessage
		err     error
		want    string
		wantErr bool
	}{
		{"simple string", json.RawMessage(`"hello"`), nil, "hello", false},
		{"empty string", json.RawMessage(`""`), nil, "", false},
		{"hex value", json.RawMessage(`"0xdead"`), nil, "0xdead", false},
		{"passthrough error", nil, errors.New("fail"), "", true},
		{"not a string", json.RawMessage(`123`), nil, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadString(tt.input, tt.err)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ReadString() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ReadString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadTo(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var result map[string]string
		err := ReadTo(&result)(json.RawMessage(`{"key":"value"}`), nil)
		if err != nil {
			t.Fatal(err)
		}
		if result["key"] != "value" {
			t.Errorf("got %v, want map with key=value", result)
		}
	})

	t.Run("passthrough error", func(t *testing.T) {
		var result string
		err := ReadTo(&result)(nil, errors.New("upstream error"))
		if err == nil || err.Error() != "upstream error" {
			t.Errorf("expected upstream error, got %v", err)
		}
	})
}

func TestReadAs(t *testing.T) {
	t.Run("struct", func(t *testing.T) {
		type block struct {
			Number string `json:"number"`
		}
		got, err := ReadAs[block](json.RawMessage(`{"number":"0x1b4"}`), nil)
		if err != nil {
			t.Fatal(err)
		}
		if got.Number != "0x1b4" {
			t.Errorf("got %q, want %q", got.Number, "0x1b4")
		}
	})

	t.Run("passthrough error", func(t *testing.T) {
		_, err := ReadAs[string](nil, errors.New("fail"))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
