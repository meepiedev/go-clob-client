package types

// Based on: py-clob-client-main/py_clob_client/constants.py
const (
	// Zero address used for public orders
	// Based on: py-clob-client-main/py_clob_client/constants.py:8
	ZeroAddress = "0x0000000000000000000000000000000000000000"
	
	// Order sides
	// Based on: py-clob-client-main/py_clob_client/order_builder/constants.py:1-2
	BUY  = "BUY"
	SELL = "SELL"
	
	// Client modes
	// Based on: py-clob-client-main/py_clob_client/constants.py:1-6
	L0 = 0 // Level 0: No authentication
	L1 = 1 // Level 1: Private key authentication
	L2 = 2 // Level 2: API key authentication
	
	// Error messages
	// Based on: py-clob-client-main/py_clob_client/constants.py:10-14
	L1AuthUnavailable = "Level 1 Authentication Unavailable"
	L2AuthUnavailable = "Level 2 Authentication Unavailable"
	
	// Cursor constants
	// Based on: py-clob-client-main/py_clob_client/constants.py:16
	EndCursor = "LTE="
	
	// Signature types (from go-order-utils)
	// Based on: go-order-utils-main/pkg/model/signature_type.go:3-7
	EOA       = 0
	POLY_PROXY = 1
	POLY_GNOSIS_SAFE = 2
)

// Token decimals
// Based on: py-clob-client-main/py_clob_client/order_builder/helpers.py:14
const TokenDecimals = 6