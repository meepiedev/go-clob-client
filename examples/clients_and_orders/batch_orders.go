package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/polymarket/go-order-utils/pkg/model"
	"github.com/pooofdevelopment/go-clob-client/pkg/client"
	"github.com/pooofdevelopment/go-clob-client/pkg/types"
)

// Example: Create and post multiple orders in a batch
// This demonstrates:
// - Creating a Level 2 client
// - Finding an active market with YES/NO tokens
// - Creating multiple orders (up to 5 per batch)
// - Posting them all in a single batch request
func main() {
	// Check for required environment variables
	privateKey := os.Getenv("PK")
	if privateKey == "" {
		log.Fatal("Please set PK environment variable with your private key")
	}

	// Optional: Proxy wallet address for POLY_PROXY signature type
	walletAddress := os.Getenv("WALLET_ADDRESS")

	// Configuration
	host := "https://clob.polymarket.com"
	chainID := 137 // Polygon mainnet

	fmt.Println("Creating client...")

	// Create client
	var clobClient *client.ClobClient
	var err error

	if walletAddress != "" {
		// Use POLY_PROXY signature type with proxy wallet
		fmt.Printf("Using proxy wallet: %s\n", walletAddress)
		sigType := model.POLY_PROXY
		clobClient, err = client.NewClobClient(host, chainID, privateKey, nil, &sigType, &walletAddress)
	} else {
		// Use standard EOA signature type
		fmt.Println("Using EOA signature type (no proxy wallet)")
		clobClient, err = client.NewClobClient(host, chainID, privateKey, nil, nil, nil)
	}

	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	fmt.Printf("✓ Client initialized\n")
	fmt.Printf("  Signer address: %s\n", clobClient.GetAddress())

	// Create/derive API credentials
	creds, err := clobClient.CreateOrDeriveApiCreds(nil)
	if err != nil {
		log.Fatal("Failed to create/derive API credentials:", err)
	}

	// Upgrade to Level 2
	clobClient.SetApiCreds(creds)
	fmt.Println("✓ Upgraded to Level 2 authentication")

	// Get active markets
	fmt.Println("\nFetching active markets...")
	activeBool := true
	closedBool := false
	gammaParams := &types.GammaMarketsParams{
		Active: &activeBool,
		Closed: &closedBool,
		Limit:  10,
	}

	markets, err := clobClient.GetGammaMarkets(gammaParams)
	if err != nil {
		log.Fatal("Failed to get gamma markets:", err)
	}

	if len(markets) == 0 {
		log.Fatal("No active markets found")
	}

	// Find a market with at least 2 token IDs (YES and NO)
	var yesTokenID, noTokenID string
	var marketQuestion string
	for _, market := range markets {
		if len(market.ClobTokenIDs) >= 2 {
			yesTokenID = market.ClobTokenIDs[0]
			noTokenID = market.ClobTokenIDs[1]
			marketQuestion = market.Question
			break
		}
	}

	if yesTokenID == "" || noTokenID == "" {
		log.Fatal("No market with YES/NO tokens found")
	}

	fmt.Printf("\nSelected market: %s\n", marketQuestion)
	fmt.Printf("YES Token ID: %s\n", yesTokenID)
	fmt.Printf("NO Token ID: %s\n", noTokenID)

	// Create multiple orders for batch submission
	fmt.Println("\nCreating batch orders...")

	// Order 1: Buy YES tokens at 0.40
	order1Args := &types.OrderArgs{
		TokenID:    yesTokenID,
		Price:      0.40,
		Size:       25.0, // $10 worth
		Side:       types.BUY,
		FeeRateBps: 0,
		Nonce:      0,
		Expiration: 0,
		Taker:      types.ZeroAddress,
	}

	// Order 2: Buy YES tokens at 0.35 (lower price)
	order2Args := &types.OrderArgs{
		TokenID:    yesTokenID,
		Price:      0.35,
		Size:       30.0, // $10.50 worth
		Side:       types.BUY,
		FeeRateBps: 0,
		Nonce:      0,
		Expiration: 0,
		Taker:      types.ZeroAddress,
	}

	// Order 3: Buy NO tokens at 0.45
	order3Args := &types.OrderArgs{
		TokenID:    noTokenID,
		Price:      0.45,
		Size:       20.0, // $9 worth
		Side:       types.BUY,
		FeeRateBps: 0,
		Nonce:      0,
		Expiration: 0,
		Taker:      types.ZeroAddress,
	}

	// Create signed orders
	signedOrder1, err := clobClient.CreateOrder(order1Args, nil)
	if err != nil {
		log.Fatal("Failed to create order 1:", err)
	}

	signedOrder2, err := clobClient.CreateOrder(order2Args, nil)
	if err != nil {
		log.Fatal("Failed to create order 2:", err)
	}

	signedOrder3, err := clobClient.CreateOrder(order3Args, nil)
	if err != nil {
		log.Fatal("Failed to create order 3:", err)
	}

	fmt.Println("✓ All orders created and signed")

	// Prepare batch order arguments
	batchOrders := []types.PostOrdersArgs{
		{
			Order:     signedOrder1,
			OrderType: types.OrderTypeGTC, // Good Till Cancelled
		},
		{
			Order:     signedOrder2,
			OrderType: types.OrderTypeGTC,
		},
		{
			Order:     signedOrder3,
			OrderType: types.OrderTypeGTC,
		},
	}

	// Display batch summary
	fmt.Println("\nBatch order summary:")
	fmt.Println("Order 1: BUY 25.00 YES @ $0.40 = $10.00")
	fmt.Println("Order 2: BUY 30.00 YES @ $0.35 = $10.50")
	fmt.Println("Order 3: BUY 20.00 NO  @ $0.45 = $9.00")
	fmt.Println("Total value: $29.50")

	// Post the batch order
	fmt.Println("\nPosting batch order...")
	response, err := clobClient.PostOrders(batchOrders)
	if err != nil {
		fmt.Printf("Error posting batch order: %v\n", err)
		fmt.Println("\nCommon errors:")
		fmt.Println("- 'not enough balance/allowance': Account needs USDC balance")
		fmt.Println("- 'invalid amount': Each order size must be >= $1.00")
		fmt.Println("- 'maximum 5 orders': Batch is limited to 5 orders")
		return
	}

	// Check response
	if !response.Success {
		fmt.Printf("Batch order failed: %s\n", response.ErrorMsg)
		return
	}

	fmt.Println("✓ Batch order posted successfully!")
	fmt.Printf("Order ID: %s\n", response.OrderID)
	if len(response.OrderHashes) > 0 {
		fmt.Println("Order hashes:")
		for i, hash := range response.OrderHashes {
			fmt.Printf("  Order %d: %s\n", i+1, hash)
		}
	}

	// Alternative: Using the convenience method
	fmt.Println("\nAlternative method using CreateAndPostOrders...")

	// Define orders with their parameters
	ordersList := []struct {
		Args      *types.OrderArgs
		Options   *types.PartialCreateOrderOptions
		OrderType types.OrderType
	}{
		{
			Args: &types.OrderArgs{
				TokenID:    yesTokenID,
				Price:      0.30,
				Size:       35.0,
				Side:       types.BUY,
				FeeRateBps: 0,
				Nonce:      0,
				Expiration: 0,
				Taker:      types.ZeroAddress,
			},
			Options:   nil,
			OrderType: types.OrderTypeFOK, // Fill or Kill
		},
		{
			Args: &types.OrderArgs{
				TokenID:    noTokenID,
				Price:      0.40,
				Size:       25.0,
				Side:       types.BUY,
				FeeRateBps: 0,
				Nonce:      0,
				Expiration: 0,
				Taker:      types.ZeroAddress,
			},
			Options:   nil,
			OrderType: types.OrderTypeFAK, // Fill and Kill
		},
	}

	// Create and post in one call
	response2, err := clobClient.CreateAndPostOrders(ordersList)
	if err != nil {
		fmt.Printf("Error with convenience method: %v\n", err)
	} else if response2.Success {
		fmt.Println("✓ Convenience method batch order posted successfully!")
		responseJSON, _ := json.MarshalIndent(response2, "", "  ")
		fmt.Println(string(responseJSON))
	}
}
