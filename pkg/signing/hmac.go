package signing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// BuildHMACSignature creates an HMAC signature for Level 2 authentication
// Based on: py-clob-client-main/py_clob_client/signing/hmac.py:6-22
func BuildHMACSignature(secret string, timestamp int64, method string, requestPath string, body interface{}) (string, error) {
	// Decode the base64 secret
	// Based on: py-clob-client-main/py_clob_client/signing/hmac.py:12
	decodedSecret, err := base64.URLEncoding.DecodeString(secret)
	if err != nil {
		return "", fmt.Errorf("failed to decode secret: %w", err)
	}
	
	// Build the message to sign
	// Based on: py-clob-client-main/py_clob_client/signing/hmac.py:13
	message := fmt.Sprintf("%d%s%s", timestamp, method, requestPath)
	
	// Add body if present
	// Based on: py-clob-client-main/py_clob_client/signing/hmac.py:14-17
	if body != nil {
		// Convert body to JSON string
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			return "", fmt.Errorf("failed to marshal body: %w", err)
		}
		
		// Replace single quotes with double quotes to match Python behavior
		// Based on: py-clob-client-main/py_clob_client/signing/hmac.py:16-17
		bodyStr := string(bodyJSON)
		bodyStr = strings.ReplaceAll(bodyStr, "'", "\"")
		message += bodyStr
	}
	
	// Create HMAC with SHA256
	// Based on: py-clob-client-main/py_clob_client/signing/hmac.py:19
	h := hmac.New(sha256.New, decodedSecret)
	h.Write([]byte(message))
	
	// Encode to base64
	// Based on: py-clob-client-main/py_clob_client/signing/hmac.py:22
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))
	
	return signature, nil
}