package tests

import (
	"github.com/pooofdevelopment/go-clob-client/pkg/utilities"
	"math/big"
	"testing"

	"github.com/pooofdevelopment/go-clob-client/pkg/types"
)

// TestToTokenDecimals tests conversion to token decimals
// Based on: py-clob-client-main/py_clob_client/order_builder/helpers.py:9-15
func TestToTokenDecimals(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		expected string
	}{
		{
			name:     "whole number",
			amount:   100,
			expected: "100000000",
		},
		{
			name:     "decimal",
			amount:   100.5,
			expected: "100500000",
		},
		{
			name:     "small decimal",
			amount:   0.000001,
			expected: "1",
		},
		{
			name:     "zero",
			amount:   0,
			expected: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utilities.ToTokenDecimals(tt.amount)
			if result.String() != tt.expected {
				t.Errorf("ToTokenDecimals(%f) = %s, want %s", tt.amount, result.String(), tt.expected)
			}
		})
	}
}

// TestFromTokenDecimals tests conversion from token decimals
func TestFromTokenDecimals(t *testing.T) {
	tests := []struct {
		name     string
		amount   *big.Int
		expected float64
	}{
		{
			name:     "whole number",
			amount:   big.NewInt(100000000),
			expected: 100,
		},
		{
			name:     "with decimal",
			amount:   big.NewInt(100500000),
			expected: 100.5,
		},
		{
			name:     "small value",
			amount:   big.NewInt(1),
			expected: 0.000001,
		},
		{
			name:     "zero",
			amount:   big.NewInt(0),
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utilities.FromTokenDecimals(tt.amount)
			if result != tt.expected {
				t.Errorf("FromTokenDecimals(%s) = %f, want %f", tt.amount.String(), result, tt.expected)
			}
		})
	}
}

// TestRoundNormal tests normal rounding
// Based on: py-clob-client-main/py_clob_client/order_builder/helpers.py:18-21
func TestRoundNormal(t *testing.T) {
	tests := []struct {
		name          string
		value         float64
		decimalPlaces int
		expected      float64
	}{
		{
			name:          "round to 2 decimals",
			value:         1.23456,
			decimalPlaces: 2,
			expected:      1.23,
		},
		{
			name:          "round up",
			value:         1.235,
			decimalPlaces: 2,
			expected:      1.24,
		},
		{
			name:          "round down",
			value:         1.234,
			decimalPlaces: 2,
			expected:      1.23,
		},
		{
			name:          "no decimals",
			value:         1.5,
			decimalPlaces: 0,
			expected:      2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utilities.RoundNormal(tt.value, tt.decimalPlaces)
			if result != tt.expected {
				t.Errorf("RoundNormal(%f, %d) = %f, want %f", tt.value, tt.decimalPlaces, result, tt.expected)
			}
		})
	}
}

// TestPriceValid tests price validation
// Based on: py-clob-client-main/py_clob_client/utilities.py:76-84
func TestPriceValid(t *testing.T) {
	tests := []struct {
		name     string
		price    float64
		tickSize string
		expected bool
	}{
		{
			name:     "valid price with 0.01 tick",
			price:    0.5,
			tickSize: "0.01",
			expected: true,
		},
		{
			name:     "invalid price - not multiple of tick",
			price:    0.555,
			tickSize: "0.01",
			expected: false,
		},
		{
			name:     "price too low",
			price:    0.005,
			tickSize: "0.01",
			expected: false,
		},
		{
			name:     "price too high",
			price:    0.995,
			tickSize: "0.01",
			expected: false,
		},
		{
			name:     "valid edge case - min price",
			price:    0.01,
			tickSize: "0.01",
			expected: true,
		},
		{
			name:     "valid edge case - max price",
			price:    0.99,
			tickSize: "0.01",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utilities.PriceValid(tt.price, tt.tickSize)
			if result != tt.expected {
				t.Errorf("PriceValid(%f, %s) = %v, want %v", tt.price, tt.tickSize, result, tt.expected)
			}
		})
	}
}

// TestIsTickSizeSmaller tests tick size comparison
// Based on: py-clob-client-main/py_clob_client/utilities.py:68-73
func TestIsTickSizeSmaller(t *testing.T) {
	tests := []struct {
		name     string
		tick1    string
		tick2    string
		expected bool
	}{
		{
			name:     "0.001 < 0.01",
			tick1:    "0.001",
			tick2:    "0.01",
			expected: true,
		},
		{
			name:     "0.01 not < 0.001",
			tick1:    "0.01",
			tick2:    "0.001",
			expected: false,
		},
		{
			name:     "equal ticks",
			tick1:    "0.01",
			tick2:    "0.01",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utilities.IsTickSizeSmaller(tt.tick1, tt.tick2)
			if result != tt.expected {
				t.Errorf("IsTickSizeSmaller(%s, %s) = %v, want %v", tt.tick1, tt.tick2, result, tt.expected)
			}
		})
	}
}

// TestParseRawOrderbookSummary tests orderbook parsing
// Based on: py-clob-client-main/py_clob_client/utilities.py:13-26
func TestParseRawOrderbookSummary(t *testing.T) {
	raw := map[string]interface{}{
		"market":    "0x1234",
		"asset_id":  "0x5678",
		"timestamp": "1234567890",
		"bids": []interface{}{
			[]interface{}{"0.4", "100"},
			[]interface{}{"0.3", "200"},
		},
		"asks": []interface{}{
			[]interface{}{"0.6", "150"},
			[]interface{}{"0.7", "250"},
		},
	}

	result, err := utilities.ParseRawOrderbookSummary(raw)
	if err != nil {
		t.Fatalf("ParseRawOrderbookSummary failed: %v", err)
	}

	if result.Market != "0x1234" {
		t.Errorf("Market = %s, want 0x1234", result.Market)
	}

	if len(result.Bids) != 2 {
		t.Errorf("Bids length = %d, want 2", len(result.Bids))
	}

	if result.Bids[0].Price != "0.4" || result.Bids[0].Size != "100" {
		t.Errorf("First bid incorrect: %+v", result.Bids[0])
	}

	if len(result.Asks) != 2 {
		t.Errorf("Asks length = %d, want 2", len(result.Asks))
	}
}

// TestGenerateOrderbookSummaryHash tests hash generation
// Based on: py-clob-client-main/py_clob_client/utilities.py:29-32
func TestGenerateOrderbookSummaryHash(t *testing.T) {
	orderbook := &types.OrderBookSummary{
		Market:    "0x1234",
		AssetID:   "0x5678",
		Timestamp: "1234567890",
		Bids: []types.OrderSummary{
			{Price: "0.4", Size: "100"},
		},
		Asks: []types.OrderSummary{
			{Price: "0.6", Size: "150"},
		},
	}

	hash := utilities.GenerateOrderbookSummaryHash(orderbook)

	// Hash should be non-empty hex string
	if len(hash) != 64 {
		t.Errorf("Hash length = %d, want 64", len(hash))
	}

	// Same orderbook should produce same hash
	hash2 := utilities.GenerateOrderbookSummaryHash(orderbook)
	if hash != hash2 {
		t.Error("Same orderbook produced different hashes")
	}
}
