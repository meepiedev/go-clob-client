package signer

import (
	"crypto/ecdsa"
	"fmt"
	
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Signer handles private key operations and signing
// Based on: py-clob-client-main/py_clob_client/signer.py:4-23
type Signer struct {
	privateKey *ecdsa.PrivateKey
	address    common.Address
	chainID    int
}

// NewSigner creates a new signer from a private key string
// Based on: py-clob-client-main/py_clob_client/signer.py:5-11
func NewSigner(privateKeyHex string, chainID int) (*Signer, error) {
	if privateKeyHex == "" || chainID == 0 {
		return nil, fmt.Errorf("private key and chain ID are required")
	}
	
	// Remove 0x prefix if present
	if len(privateKeyHex) >= 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}
	
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to cast public key to ECDSA")
	}
	
	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	
	return &Signer{
		privateKey: privateKey,
		address:    address,
		chainID:    chainID,
	}, nil
}

// Address returns the signer's address
// Based on: py-clob-client-main/py_clob_client/signer.py:12-13
func (s *Signer) Address() string {
	return s.address.Hex()
}

// GetChainID returns the chain ID
// Based on: py-clob-client-main/py_clob_client/signer.py:15-16
func (s *Signer) GetChainID() int {
	return s.chainID
}

// Sign signs a message hash
// Based on: py-clob-client-main/py_clob_client/signer.py:18-23
// Also uses: go-order-utils-main/pkg/signer/signer.go:12-19
func (s *Signer) Sign(messageHash []byte) (string, error) {
	// Convert to hash if needed
	var hash common.Hash
	if len(messageHash) == 32 {
		hash = common.BytesToHash(messageHash)
	} else {
		return "", fmt.Errorf("invalid message hash length: expected 32, got %d", len(messageHash))
	}
	
	// Sign the hash
	signature, err := crypto.Sign(hash.Bytes(), s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %w", err)
	}
	
	// Transform V from 0/1 to 27/28 (Ethereum convention)
	// Based on: go-order-utils-main/pkg/signer/signer.go:17
	signature[64] += 27
	
	// Return as hex string with 0x prefix
	return "0x" + common.Bytes2Hex(signature), nil
}

// GetPrivateKey returns the private key (for internal use)
func (s *Signer) GetPrivateKey() *ecdsa.PrivateKey {
	return s.privateKey
}