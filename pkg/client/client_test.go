package client

import (
	"testing"
	
	"github.com/pooofdevelopment/go-clob-client/pkg/types"
)

// TestNewClobClient tests creating a new CLOB client
// Based on testing patterns from py-clob-client-main and go-order-utils-main
func TestNewClobClient(t *testing.T) {
	tests := []struct {
		name        string
		host        string
		chainID     int
		privateKey  string
		creds       *types.ApiCreds
		wantMode    int
		wantErr     bool
	}{
		{
			name:        "Level 0 client - no auth",
			host:        "https://clob.polymarket.com",
			chainID:     137,
			privateKey:  "",
			creds:       nil,
			wantMode:    types.L0,
			wantErr:     false,
		},
		{
			name:        "Level 1 client - private key only",
			host:        "https://clob.polymarket.com/",
			chainID:     137,
			privateKey:  "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			creds:       nil,
			wantMode:    types.L1,
			wantErr:     false,
		},
		{
			name:        "Level 2 client - full auth",
			host:        "https://clob.polymarket.com",
			chainID:     137,
			privateKey:  "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			creds: &types.ApiCreds{
				ApiKey:        "test-key",
				ApiSecret:     "test-secret",
				ApiPassphrase: "test-passphrase",
			},
			wantMode:    types.L2,
			wantErr:     false,
		},
		{
			name:        "Invalid private key",
			host:        "https://clob.polymarket.com",
			chainID:     137,
			privateKey:  "invalid-key",
			creds:       nil,
			wantMode:    0,
			wantErr:     true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClobClient(tt.host, tt.chainID, tt.privateKey, tt.creds, nil, nil)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClobClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if err == nil {
				if client.mode != tt.wantMode {
					t.Errorf("NewClobClient() mode = %v, want %v", client.mode, tt.wantMode)
				}
				
				// Check host normalization
				expectedHost := tt.host
				if expectedHost[len(expectedHost)-1] == '/' {
					expectedHost = expectedHost[:len(expectedHost)-1]
				}
				if client.host != expectedHost {
					t.Errorf("NewClobClient() host = %v, want %v", client.host, expectedHost)
				}
			}
		})
	}
}

// TestGetClientMode tests the client mode determination
func TestGetClientMode(t *testing.T) {
	// Test L0 mode
	client := &ClobClient{}
	if mode := client.getClientMode(); mode != types.L0 {
		t.Errorf("getClientMode() = %v, want %v", mode, types.L0)
	}
	
	// Test L1 mode - would need a mock signer
	// Test L2 mode - would need mock signer and creds
}

// TestAssertAuth tests authentication assertions
func TestAssertAuth(t *testing.T) {
	// L0 client
	client := &ClobClient{mode: types.L0}
	
	if err := client.assertLevel1Auth(); err == nil {
		t.Error("assertLevel1Auth() should fail for L0 client")
	}
	
	if err := client.assertLevel2Auth(); err == nil {
		t.Error("assertLevel2Auth() should fail for L0 client")
	}
	
	// L1 client
	client.mode = types.L1
	
	if err := client.assertLevel1Auth(); err != nil {
		t.Error("assertLevel1Auth() should succeed for L1 client")
	}
	
	if err := client.assertLevel2Auth(); err == nil {
		t.Error("assertLevel2Auth() should fail for L1 client")
	}
	
	// L2 client
	client.mode = types.L2
	
	if err := client.assertLevel1Auth(); err != nil {
		t.Error("assertLevel1Auth() should succeed for L2 client")
	}
	
	if err := client.assertLevel2Auth(); err != nil {
		t.Error("assertLevel2Auth() should succeed for L2 client")
	}
}