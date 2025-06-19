package tests

import (
	"github.com/pooofdevelopment/go-clob-client/pkg/signing"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pooofdevelopment/go-clob-client/pkg/signer"
)

func TestSignClobAuthMessage(t *testing.T) {
	// Test with known values
	testPrivateKey := "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	expectedAddress := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

	s, err := signer.NewSigner(testPrivateKey, 137)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	if s.Address() != expectedAddress {
		t.Errorf("Address mismatch: got %s, want %s", s.Address(), expectedAddress)
	}

	// Test signing
	timestamp := int64(1234567890)
	nonce := 0

	signature, err := signing.SignClobAuthMessage(s, timestamp, nonce)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	// Signature should be hex with 0x prefix
	if len(signature) != 132 { // 0x + 130 hex chars (65 bytes)
		t.Errorf("Signature length wrong: got %d, want 132", len(signature))
	}

	// Check if it's valid hex
	if signature[:2] != "0x" {
		t.Error("Signature should start with 0x")
	}

	// Try to decode it
	sigBytes := common.FromHex(signature)
	if len(sigBytes) != 65 {
		t.Errorf("Decoded signature length wrong: got %d, want 65", len(sigBytes))
	}
}
