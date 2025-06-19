package errors

import "fmt"

// PolyException represents a custom exception for the CLOB client
// Based on: py-clob-client-main/py_clob_client/exceptions.py:1-5
type PolyException struct {
	Message string
}

func (e *PolyException) Error() string {
	return e.Message
}

// NewPolyException creates a new PolyException
func NewPolyException(message string) *PolyException {
	return &PolyException{Message: message}
}

// Common errors
var (
	// Based on: py-clob-client-main/py_clob_client/constants.py:10-14
	ErrL1AuthUnavailable = NewPolyException("Level 1 Authentication Unavailable")
	ErrL2AuthUnavailable = NewPolyException("Level 2 Authentication Unavailable")
	
	// Additional common errors inferred from Python client usage
	ErrInvalidChainID    = NewPolyException("Invalid chain ID")
	ErrInvalidTickSize   = NewPolyException("Invalid tick size")
	ErrInvalidPrice      = NewPolyException("Invalid price")
	ErrNoOrderbook       = NewPolyException("No orderbook available")
	ErrNoMatch           = NewPolyException("No match found")
)

// NewInvalidTickSizeError creates a tick size validation error
// Based on: py-clob-client-main/py_clob_client/client.py:325-332
func NewInvalidTickSizeError(tickSize, minTickSize string) error {
	return fmt.Errorf("invalid tick size (%s), minimum for the market is %s", tickSize, minTickSize)
}

// NewInvalidPriceError creates a price validation error
// Based on: py-clob-client-main/py_clob_client/client.py:352-359
func NewInvalidPriceError(price float64, minTickSize, maxPrice string) error {
	return fmt.Errorf("price (%f), min: %s - max: %s", price, minTickSize, maxPrice)
}