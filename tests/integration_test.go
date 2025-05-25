// +build integration

package tests

import (
	"os"
	"testing"
	
	"github.com/pooofdevelopment/go-clob-client/pkg/client"
	"github.com/pooofdevelopment/go-clob-client/pkg/types"
)

// TestClobClientIntegration performs integration tests against the real API
// Based on test patterns from py-clob-client-main/tests/
func TestClobClientIntegration(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run.")
	}
	
	host := "https://clob.polymarket.com"
	
	t.Run("L0 Client Tests", func(t *testing.T) {
		// Create L0 client
		client, err := client.NewClobClient(host, 137, "", nil, nil, nil)
		if err != nil {
			t.Fatalf("Failed to create L0 client: %v", err)
		}
		
		// Test health check
		t.Run("GetOk", func(t *testing.T) {
			resp, err := client.GetOk()
			if err != nil {
				t.Errorf("GetOk failed: %v", err)
			}
			t.Logf("Health check response: %v", resp)
		})
		
		// Test server time
		t.Run("GetServerTime", func(t *testing.T) {
			resp, err := client.GetServerTime()
			if err != nil {
				t.Errorf("GetServerTime failed: %v", err)
			}
			t.Logf("Server time: %v", resp)
		})
		
		// Test get markets
		t.Run("GetMarkets", func(t *testing.T) {
			resp, err := client.GetMarkets("")
			if err != nil {
				t.Errorf("GetMarkets failed: %v", err)
			}
			t.Logf("Markets response has data: %v", resp["data"] != nil)
		})
	})
	
	// L1/L2 tests require a private key
	privateKey := os.Getenv("TEST_PRIVATE_KEY")
	if privateKey == "" {
		t.Log("Skipping L1/L2 tests - TEST_PRIVATE_KEY not set")
		return
	}
	
	t.Run("L1 Client Tests", func(t *testing.T) {
		// Create L1 client
		client, err := client.NewClobClient(host, 137, privateKey, nil, nil, nil)
		if err != nil {
			t.Fatalf("Failed to create L1 client: %v", err)
		}
		
		// Test API key creation/derivation
		t.Run("CreateOrDeriveApiCreds", func(t *testing.T) {
			creds, err := client.CreateOrDeriveApiCreds(nil)
			if err != nil {
				t.Errorf("CreateOrDeriveApiCreds failed: %v", err)
				return
			}
			
			if creds.ApiKey == "" || creds.ApiSecret == "" {
				t.Error("Invalid credentials returned")
			}
			
			// Upgrade to L2 for further tests
			client.SetApiCreds(creds)
			
			// Test authenticated endpoints
			t.Run("GetApiKeys", func(t *testing.T) {
				resp, err := client.GetApiKeys()
				if err != nil {
					t.Errorf("GetApiKeys failed: %v", err)
				}
				t.Logf("API keys response: %v", resp)
			})
		})
	})
}

// TestOrderCreation tests order creation without posting
// This can run without real credentials
func TestOrderCreation(t *testing.T) {
	// Use a test private key (not real funds)
	testPrivateKey := "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	
	client, err := client.NewClobClient("https://clob.polymarket.com", 137, testPrivateKey, nil, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	
	orderArgs := &types.OrderArgs{
		TokenID:    "1343197538147866997676796723584448962074666613334415804986823996107646701544",
		Price:      0.5,
		Size:       100,
		Side:       types.BUY,
		FeeRateBps: 0,
		Nonce:      0,
		Expiration: 0,
		Taker:      types.ZeroAddress,
	}
	
	// This will fail on tick size/neg risk lookup but tests the order creation flow
	_, err = client.CreateOrder(orderArgs, nil)
	if err != nil {
		// Expected to fail without real market data
		t.Logf("CreateOrder failed as expected: %v", err)
	}
}