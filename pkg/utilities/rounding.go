package utilities

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// RoundNormal performs standard rounding to specified decimal places
// Based on: py-clob-client-main/py_clob_client/order_builder/helpers.py:18-21
func RoundNormal(value float64, decimalPlaces int) float64 {
	multiplier := math.Pow(10, float64(decimalPlaces))
	return math.Round(value*multiplier) / multiplier
}

// RoundDown performs floor rounding to specified decimal places
// Based on: py-clob-client-main/py_clob_client/order_builder/helpers.py:24-27
func RoundDown(value float64, decimalPlaces int) float64 {
	multiplier := math.Pow(10, float64(decimalPlaces))
	return math.Floor(value*multiplier) / multiplier
}

// RoundUp performs ceiling rounding to specified decimal places
// Based on: py-clob-client-main/py_clob_client/order_builder/helpers.py:30-33
func RoundUp(value float64, decimalPlaces int) float64 {
	multiplier := math.Pow(10, float64(decimalPlaces))
	return math.Ceil(value*multiplier) / multiplier
}

// DecimalPlaces returns the number of decimal places in a float
// Based on: py-clob-client-main/py_clob_client/order_builder/helpers.py:36-39
func DecimalPlaces(value float64) int {
	// Convert to string to count decimal places
	str := fmt.Sprintf("%f", value)
	
	// Remove trailing zeros
	str = strings.TrimRight(str, "0")
	str = strings.TrimRight(str, ".")
	
	// Find decimal point
	parts := strings.Split(str, ".")
	if len(parts) < 2 {
		return 0
	}
	
	return len(parts[1])
}

// ParseFloat safely parses a string to float64
func ParseFloat(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}
	return strconv.ParseFloat(s, 64)
}