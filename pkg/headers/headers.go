package headers

import (
	"fmt"
	"time"
	
	"github.com/pooofdevelopment/go-clob-client/pkg/signer"
	"github.com/pooofdevelopment/go-clob-client/pkg/signing"
	"github.com/pooofdevelopment/go-clob-client/pkg/types"
)

// Header constants
// Based on: py-clob-client-main/py_clob_client/headers/headers.py:7-12
const (
	POLY_ADDRESS    = "POLY_ADDRESS"
	POLY_SIGNATURE  = "POLY_SIGNATURE"
	POLY_TIMESTAMP  = "POLY_TIMESTAMP"
	POLY_NONCE      = "POLY_NONCE"
	POLY_API_KEY    = "POLY_API_KEY"
	POLY_PASSPHRASE = "POLY_PASSPHRASE"
)

// CreateLevel1Headers creates Level 1 Poly headers for a request
// Based on: py-clob-client-main/py_clob_client/headers/headers.py:15-33
func CreateLevel1Headers(signer *signer.Signer, nonce *int) (map[string]string, error) {
	// Get current timestamp
	// Based on: py-clob-client-main/py_clob_client/headers/headers.py:19
	timestamp := time.Now().Unix()
	
	// Default nonce to 0 if not provided
	// Based on: py-clob-client-main/py_clob_client/headers/headers.py:21-23
	n := 0
	if nonce != nil {
		n = *nonce
	}
	
	// Sign the authentication message
	// Based on: py-clob-client-main/py_clob_client/headers/headers.py:25
	signature, err := signing.SignClobAuthMessage(signer, timestamp, n)
	if err != nil {
		return nil, fmt.Errorf("failed to sign auth message: %w", err)
	}
	
	// Create headers
	// Based on: py-clob-client-main/py_clob_client/headers/headers.py:26-31
	headers := map[string]string{
		POLY_ADDRESS:   signer.Address(),
		POLY_SIGNATURE: signature,
		POLY_TIMESTAMP: fmt.Sprintf("%d", timestamp),
		POLY_NONCE:     fmt.Sprintf("%d", n),
	}
	
	return headers, nil
}

// CreateLevel2Headers creates Level 2 Poly headers for a request
// Based on: py-clob-client-main/py_clob_client/headers/headers.py:36-56
func CreateLevel2Headers(signer *signer.Signer, creds *types.ApiCreds, requestArgs *types.RequestArgs) (map[string]string, error) {
	// Get current timestamp
	// Based on: py-clob-client-main/py_clob_client/headers/headers.py:40
	timestamp := time.Now().Unix()
	
	// Build HMAC signature
	// Based on: py-clob-client-main/py_clob_client/headers/headers.py:42-48
	hmacSig, err := signing.BuildHMACSignature(
		creds.ApiSecret,
		timestamp,
		requestArgs.Method,
		requestArgs.RequestPath,
		requestArgs.Body,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build HMAC signature: %w", err)
	}
	
	// Create headers
	// Based on: py-clob-client-main/py_clob_client/headers/headers.py:50-56
	headers := map[string]string{
		POLY_ADDRESS:    signer.Address(),
		POLY_SIGNATURE:  hmacSig,
		POLY_TIMESTAMP:  fmt.Sprintf("%d", timestamp),
		POLY_API_KEY:    creds.ApiKey,
		POLY_PASSPHRASE: creds.ApiPassphrase,
	}
	
	return headers, nil
}