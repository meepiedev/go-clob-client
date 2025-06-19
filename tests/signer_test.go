package tests

import (
	signer2 "github.com/pooofdevelopment/go-clob-client/pkg/signer"
	"strings"
	"testing"
)

// TestNewSigner tests signer creation
// Based on: py-clob-client-main/py_clob_client/signer.py:5-11
func TestNewSigner(t *testing.T) {
	tests := []struct {
		name       string
		privateKey string
		chainID    int
		wantErr    bool
		wantAddr   string
	}{
		{
			name:       "valid private key with 0x prefix",
			privateKey: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
			chainID:    137,
			wantErr:    false,
			wantAddr:   "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", // Known address for this test key
		},
		{
			name:       "valid private key without 0x prefix",
			privateKey: "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
			chainID:    137,
			wantErr:    false,
			wantAddr:   "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
		},
		{
			name:       "empty private key",
			privateKey: "",
			chainID:    137,
			wantErr:    true,
		},
		{
			name:       "invalid private key",
			privateKey: "invalid",
			chainID:    137,
			wantErr:    true,
		},
		{
			name:       "zero chain ID",
			privateKey: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
			chainID:    0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer, err := signer2.NewSigner(tt.privateKey, tt.chainID)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewSigner() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if signer.Address() != tt.wantAddr {
					t.Errorf("Address() = %v, want %v", signer.Address(), tt.wantAddr)
				}

				if signer.GetChainID() != tt.chainID {
					t.Errorf("GetChainID() = %v, want %v", signer.GetChainID(), tt.chainID)
				}
			}
		})
	}
}

// TestSign tests message signing
// Based on: py-clob-client-main/py_clob_client/signer.py:18-23
func TestSign(t *testing.T) {
	// Use test private key
	privateKey := "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	s, err := signer2.NewSigner(privateKey, 137)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	// Test message hash (32 bytes)
	messageHash := make([]byte, 32)
	for i := range messageHash {
		messageHash[i] = byte(i)
	}

	signature, err := s.Sign(messageHash)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}

	// Signature should be hex string with 0x prefix
	if !strings.HasPrefix(signature, "0x") {
		t.Error("Signature should have 0x prefix")
	}

	// Signature should be 132 characters (0x + 130 hex chars for 65 bytes)
	if len(signature) != 132 {
		t.Errorf("Signature length = %d, want 132", len(signature))
	}

	// Test invalid message hash length
	invalidHash := make([]byte, 31)
	_, err = s.Sign(invalidHash)
	if err == nil {
		t.Error("Sign() should fail with invalid hash length")
	}
}

// TestSignerMethods tests getter methods
func TestSignerMethods(t *testing.T) {
	privateKey := "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	chainID := 137

	s, err := signer2.NewSigner(privateKey, chainID)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	// Test Address()
	addr := s.Address()
	if addr == "" {
		t.Error("Address() returned empty string")
	}
	if !strings.HasPrefix(addr, "0x") {
		t.Error("Address should have 0x prefix")
	}
	if len(addr) != 42 {
		t.Errorf("Address length = %d, want 42", len(addr))
	}

	// Test GetChainID()
	if s.GetChainID() != chainID {
		t.Errorf("GetChainID() = %d, want %d", s.GetChainID(), chainID)
	}

	// Test GetPrivateKey()
	if s.GetPrivateKey() == nil {
		t.Error("GetPrivateKey() returned nil")
	}
}
