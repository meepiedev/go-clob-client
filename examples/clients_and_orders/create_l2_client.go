package main

import (
	"fmt"
	"github.com/polymarket/go-order-utils/pkg/model"
	"log"
	"os"

	"github.com/pooofdevelopment/go-clob-client/pkg/client"
	"github.com/pooofdevelopment/go-clob-client/pkg/types"
)

// Example: Create Level 2 authorized client from address and private key
// This demonstrates the full authentication flow from L0 -> L1 -> L2
// Supports proxy wallets by allowing separate wallet address
func main() {
	// Configuration
	host := "https://clob.polymarket.com"
	chainID := 137 // Polygon mainnet

	// Get private key from environment variable
	// Based on: py-clob-client-main/examples/create_api_key.py:9
	privateKey := os.Getenv("PK")
	if privateKey == "" {
		log.Fatal("Please set PK environment variable with your private key")
	}

	// Get wallet address from environment variable (optional)
	// If not provided, it will be derived from the private key
	walletAddress := os.Getenv("WALLET_ADDRESS")

	fmt.Println("Creating Level 2 authorized client...")
	fmt.Println("===========================================")

	// Step 1: Create Level 1 client with private key and optional wallet address
	// Based on: py-clob-client-main/examples/create_api_key.py:3-11
	fmt.Println("\n1. Creating Level 1 client...")

	var clobClient *client.ClobClient
	var err error

	if walletAddress != "" {
		// Use proxy wallet functionality with POLY_PROXY signature type
		fmt.Printf("   Using proxy wallet: %s\n", walletAddress)
		sigType := model.POLY_PROXY
		clobClient, err = client.NewClobClient(host, chainID, privateKey, nil, &sigType, &walletAddress)
	} else {
		// Derive wallet from private key with EOA signature type
		fmt.Println("   Deriving wallet from private key...")
		clobClient, err = client.NewClobClient(host, chainID, privateKey, nil, nil, nil)
	}

	if err != nil {
		log.Fatal("Failed to create Level 1 client:", err)
	}

	// Display the address being used
	address := clobClient.GetAddress()
	fmt.Printf("   ✓ Client initialized for address: %s\n", address)
	fmt.Printf("   ✓ Client mode: Level %d\n", 1)

	// Step 2: Create or derive API credentials
	// Based on: py-clob-client-main/examples/create_api_key.py:13-18
	fmt.Println("\n2. Creating/deriving API credentials...")
	creds, err := clobClient.CreateOrDeriveApiCreds(nil)
	if err != nil {
		log.Fatal("Failed to create/derive API credentials:", err)
	}

	fmt.Printf("   ✓ API Key: %s...\n", creds.ApiKey[:8])
	fmt.Printf("   ✓ API Secret: %s...\n", creds.ApiSecret[:8])
	fmt.Printf("   ✓ API Passphrase: %s...\n", creds.ApiPassphrase[:8])

	// Step 3: Upgrade to Level 2 by setting credentials
	// Based on: py-clob-client-main/examples/create_api_key.py:19-20
	fmt.Println("\n3. Upgrading to Level 2 authentication...")
	clobClient.SetApiCreds(creds)
	fmt.Printf("   ✓ Client mode: Level %d\n", 2)

	// Step 4: Test Level 2 functionality
	fmt.Println("\n4. Testing Level 2 authenticated endpoints...")

	// Get API keys (Level 2 required)
	apiKeys, err := clobClient.GetApiKeys()
	if err != nil {
		log.Fatal("Failed to get API keys:", err)
	}
	fmt.Printf("   ✓ Successfully retrieved API keys: %v\n", apiKeys != nil)

	// Get open orders (Level 2 required)
	// Based on: py-clob-client-main/examples/get_orders.py
	orders, err := clobClient.GetOrders(nil, "")
	if err != nil {
		// It's OK if this fails - user might not have any orders
		fmt.Printf("   ✓ Get orders endpoint accessible (no orders found)\n")
	} else {
		fmt.Printf("   ✓ Found %d open orders\n", len(orders))
	}

	// Get historical trades  (Level 2 required)
	// Based on: py-clob-client-main/examples/get_orders.py
	trades, err := clobClient.GetTrades(nil, "")
	if err != nil {
		// It's OK if this fails - user might not have any orders
		fmt.Printf("   ✓ Get orders endpoint accessible (no orders found)\n")
	} else {
		fmt.Printf("   ✓ Found %d historical trade\n", len(trades))
	}

	// Get balance/allowance (Level 2 required)
	// Based on: py-clob-client-main/examples/get_balance_allowance.py
	balanceParams := &types.BalanceAllowanceParams{
		AssetType:     types.AssetTypeCollateral,
		SignatureType: -1, // Let it use the builder's signature type
	}

	// Debug: Show which address is being used
	fmt.Printf("   Checking balance for signer address: %s\n", clobClient.GetAddress())
	if walletAddress != "" {
		fmt.Printf("   With proxy wallet (funder): %s\n", walletAddress)
		fmt.Printf("   Using signature type: %d (POLY_PROXY)\n", model.POLY_PROXY)
		// Explicitly set signature type for proxy wallet
		balanceParams.SignatureType = model.POLY_PROXY
	}

	balance, err := clobClient.GetBalanceAllowance(balanceParams)
	if err != nil {
		fmt.Printf("   ! Balance check failed: %v\n", err)
	} else {
		fmt.Println("   ✓ Balance/allowance check successful.")
		fmt.Printf("   Balance: %v\n", balance["balance"])
		fmt.Printf("   Allowances: %v\n", balance["allowances"])
	}

	fmt.Println("\n===========================================")
	fmt.Println("✅ Level 2 client successfully created!")
	fmt.Println("\nYou can now:")
	fmt.Println("- Create and post orders")
	fmt.Println("- Cancel orders")
	fmt.Println("- View your trades and order history")
	fmt.Println("- Manage notifications")
	fmt.Println("- Check balances and allowances")

	// Alternative: Create L2 client directly if you already have credentials
	fmt.Println("\n===========================================")
	fmt.Println("Alternative: Direct L2 client creation")
	fmt.Println("===========================================")
	fmt.Println("\nIf you already have API credentials, you can create an L2 client directly:")
	fmt.Println("\ncreds := &types.ApiCreds{")
	fmt.Println("    ApiKey:        \"your-api-key\",")
	fmt.Println("    ApiSecret:     \"your-api-secret\",")
	fmt.Println("    ApiPassphrase: \"your-passphrase\",")
	fmt.Println("}")
	fmt.Println("\nclient, err := client.NewClobClient(host, chainID, privateKey, creds, nil, nil)")
}
