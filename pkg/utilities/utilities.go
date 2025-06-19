package utilities

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	
	"github.com/pooofdevelopment/go-clob-client/pkg/types"
)

// ParseRawOrderbookSummary parses raw orderbook data into OrderBookSummary
// Based on: py-clob-client-main/py_clob_client/utilities.py:13-26
func ParseRawOrderbookSummary(raw map[string]interface{}) (*types.OrderBookSummary, error) {
	obs := &types.OrderBookSummary{}
	
	if market, ok := raw["market"].(string); ok {
		obs.Market = market
	}
	if assetID, ok := raw["asset_id"].(string); ok {
		obs.AssetID = assetID
	}
	if timestamp, ok := raw["timestamp"].(string); ok {
		obs.Timestamp = timestamp
	}
	
	// Parse bids
	if bidsRaw, ok := raw["bids"].([]interface{}); ok {
		obs.Bids = make([]types.OrderSummary, len(bidsRaw))
		for i, bidRaw := range bidsRaw {
			// Try parsing as object with price/size fields first
			if bid, ok := bidRaw.(map[string]interface{}); ok {
				price := ""
				size := ""
				if p, ok := bid["price"].(string); ok {
					price = p
				}
				if s, ok := bid["size"].(string); ok {
					size = s
				}
				obs.Bids[i] = types.OrderSummary{
					Price: price,
					Size:  size,
				}
			} else if bid, ok := bidRaw.([]interface{}); ok && len(bid) >= 2 {
				// Fallback to array format [price, size]
				obs.Bids[i] = types.OrderSummary{
					Price: fmt.Sprintf("%v", bid[0]),
					Size:  fmt.Sprintf("%v", bid[1]),
				}
			}
		}
	}
	
	// Parse asks
	if asksRaw, ok := raw["asks"].([]interface{}); ok {
		obs.Asks = make([]types.OrderSummary, len(asksRaw))
		for i, askRaw := range asksRaw {
			// Try parsing as object with price/size fields first
			if ask, ok := askRaw.(map[string]interface{}); ok {
				price := ""
				size := ""
				if p, ok := ask["price"].(string); ok {
					price = p
				}
				if s, ok := ask["size"].(string); ok {
					size = s
				}
				obs.Asks[i] = types.OrderSummary{
					Price: price,
					Size:  size,
				}
			} else if ask, ok := askRaw.([]interface{}); ok && len(ask) >= 2 {
				// Fallback to array format [price, size]
				obs.Asks[i] = types.OrderSummary{
					Price: fmt.Sprintf("%v", ask[0]),
					Size:  fmt.Sprintf("%v", ask[1]),
				}
			}
		}
	}
	
	return obs, nil
}

// GenerateOrderbookSummaryHash generates a hash for the orderbook
// Based on: py-clob-client-main/py_clob_client/utilities.py:29-32
func GenerateOrderbookSummaryHash(orderbook *types.OrderBookSummary) string {
	// Convert orderbook to JSON with compact formatting
	data, _ := json.Marshal(orderbook)
	
	// Generate SHA256 hash
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// OrderToJSON converts an order to JSON format for API submission
// Based on: py-clob-client-main/py_clob_client/utilities.py:35-65
func OrderToJSON(order interface{}, owner string, orderType types.OrderType) map[string]interface{} {
	// This will be implemented when we have the full order structure
	// For now, return a placeholder
	return map[string]interface{}{
		"order": order,
		"owner": owner,
		"orderType": string(orderType),
	}
}

// IsTickSizeSmaller checks if tick1 is smaller than tick2
// Based on: py-clob-client-main/py_clob_client/utilities.py:68-73
func IsTickSizeSmaller(tick1, tick2 string) bool {
	t1, _ := strconv.ParseFloat(tick1, 64)
	t2, _ := strconv.ParseFloat(tick2, 64)
	return t1 < t2
}

// PriceValid checks if a price is valid for the given tick size
// Based on: py-clob-client-main/py_clob_client/utilities.py:76-84
func PriceValid(price float64, tickSize string) bool {
	tick, _ := strconv.ParseFloat(tickSize, 64)
	
	// Price must be >= tick size and <= 1 - tick size
	if price < tick || price > (1 - tick) {
		return false
	}
	
	// Check if price is a multiple of tick size
	// Scale to avoid floating point issues
	tickDecimals := getDecimalPlaces(tickSize)
	scale := math.Pow(10, float64(tickDecimals))
	
	// Scale the price and tick
	scaledPrice := price * scale
	scaledTick := tick * scale
	
	// Round to nearest integer to handle floating point precision
	// Then check if price is divisible by tick
	roundedPrice := math.Round(scaledPrice)
	roundedTick := math.Round(scaledTick)
	
	if roundedTick == 0 {
		return false
	}
	
	// Check if the remainder is 0
	remainder := int64(roundedPrice) % int64(roundedTick)
	
	// Also check that rounding didn't change the price significantly
	// This prevents 0.555 from being treated as 0.56
	priceDiff := math.Abs(scaledPrice - roundedPrice)
	
	return remainder == 0 && priceDiff < 0.01
}

// getDecimalPlaces returns the number of decimal places in a string number
func getDecimalPlaces(s string) int {
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return 0
	}
	return len(parts[1])
}

// ToTokenDecimals converts a float amount to token decimals (6 decimals for USDC)
// Based on: py-clob-client-main/py_clob_client/order_builder/helpers.py:9-15
// Compatible with go-order-utils which expects string amounts already in token decimals
func ToTokenDecimals(amount float64) *big.Int {
	// Multiply by 10^6 for 6 decimal places
	// Round to avoid floating point precision issues
	scaled := amount * 1000000
	rounded := math.Round(scaled)
	
	// Handle very small numbers that might round to 0
	if rounded == 0 && amount > 0 {
		rounded = 1
	}
	
	return big.NewInt(int64(rounded))
}

// FromTokenDecimals converts from token decimals to float
// Inverse of ToTokenDecimals
func FromTokenDecimals(amount *big.Int) float64 {
	divisor := new(big.Float).SetInt(big.NewInt(1000000))
	amountFloat := new(big.Float).SetInt(amount)
	result := new(big.Float).Quo(amountFloat, divisor)
	
	floatResult, _ := result.Float64()
	return floatResult
}