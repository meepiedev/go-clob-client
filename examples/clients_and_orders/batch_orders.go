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
// - Finding a specific negRisk event by slug
// - Creating multiple orders for different candidate markets (up to 5 per batch)
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

	// Get specific event by slug
	fmt.Println("\nFetching NYC mayor nomination event...")
	eventSlug := "who-will-win-dem-nomination-for-nyc-mayor"
	eventParams := &types.GammaEventsParams{
		Slug: eventSlug,
	}

	events, err := clobClient.GetGammaEvents(eventParams)
	if err != nil {
		log.Fatal("Failed to get gamma events:", err)
	}

	if len(events) == 0 {
		log.Fatalf("Event with slug '%s' not found", eventSlug)
	}

	event := events[0]
	
	// Verify it's a negRisk event
	if !event.NegRisk {
		fmt.Printf("Warning: Event is not a negRisk event (negRisk=%v)\n", event.NegRisk)
	} else {
		fmt.Printf("✓ Confirmed negRisk event\n")
	}

	fmt.Printf("\nSelected event: %s\n", event.Title)
	fmt.Printf("Event slug: %s\n", event.Slug)
	fmt.Printf("Number of markets in event: %d\n", len(event.Markets))
	
	// NegRisk events have multiple markets, each representing a candidate
	if len(event.Markets) < 3 {
		log.Fatalf("Expected at least 3 markets in this event, found %d", len(event.Markets))
	}
	
	// Check if markets have volume data already
	marketsWithVolume := make([]types.GammaMarket, 0)
	for _, market := range event.Markets {
		if market.Volume > 0 {
			marketsWithVolume = append(marketsWithVolume, market)
		}
	}
	
	// If no volume data in event response, try to fetch individual market data
	if len(marketsWithVolume) == 0 {
		fmt.Println("\nNo volume data in event response. Fetching individual market data...")
		
		// For negRisk events, we might need to get the parent event's slug
		// and fetch aggregated data differently
		// For now, let's just use the first 3 markets with the highest token count
		if len(event.Markets) >= 3 {
			// Copy markets
			marketsWithVolume = append(marketsWithVolume, event.Markets[:3]...)
		}
	}
	
	if len(marketsWithVolume) < 3 {
		log.Fatalf("Need at least 3 markets to proceed, found %d", len(marketsWithVolume))
	}
	
	// Sort markets by volume if available (highest first)
	for i := 0; i < len(marketsWithVolume)-1; i++ {
		for j := i + 1; j < len(marketsWithVolume); j++ {
			if marketsWithVolume[j].Volume > marketsWithVolume[i].Volume {
				marketsWithVolume[i], marketsWithVolume[j] = marketsWithVolume[j], marketsWithVolume[i]
			}
		}
	}
	
	// Select top 3 markets
	market1 := marketsWithVolume[0]
	market2 := marketsWithVolume[1]
	market3 := marketsWithVolume[2]
	
	fmt.Printf("\nSelected markets:\n")
	if market1.Volume > 0 {
		fmt.Printf("1. %s - Volume: $%.2f\n", market1.Title, market1.Volume)
	} else {
		fmt.Printf("1. %s\n", market1.Title)
	}
	if market2.Volume > 0 {
		fmt.Printf("2. %s - Volume: $%.2f\n", market2.Title, market2.Volume)
	} else {
		fmt.Printf("2. %s\n", market2.Title)
	}
	if market3.Volume > 0 {
		fmt.Printf("3. %s - Volume: $%.2f\n", market3.Title, market3.Volume)
	} else {
		fmt.Printf("3. %s\n", market3.Title)
	}
	
	// Each market in a negRisk event typically has a single token
	var tokenID1, tokenID2, tokenID3 string
	
	if len(market1.ClobTokenIDs) > 0 {
		tokenID1 = market1.ClobTokenIDs[0]
	} else {
		log.Fatal("Market 1 has no token IDs")
	}
	
	if len(market2.ClobTokenIDs) > 0 {
		tokenID2 = market2.ClobTokenIDs[0]
	} else {
		log.Fatal("Market 2 has no token IDs")
	}
	
	if len(market3.ClobTokenIDs) > 0 {
		tokenID3 = market3.ClobTokenIDs[0]
	} else {
		log.Fatal("Market 3 has no token IDs")
	}
	
	fmt.Printf("\nSelected candidates:\n")
	fmt.Printf("  Entry 1: %s (Token: %s)\n", market1.Title, tokenID1)
	fmt.Printf("  Entry 2: %s (Token: %s)\n", market2.Title, tokenID2)
	fmt.Printf("  Entry 3: %s (Token: %s)\n", market3.Title, tokenID3)

	// Create multiple orders for batch submission
	fmt.Println("\nCreating batch orders...")

	// Order 1: Buy small amount of highest volume market
	order1Args := &types.OrderArgs{
		TokenID:    tokenID1,
		Price:      0.05,  // Low price for testing
		Size:       20.0,  // $1 worth at 0.05
		Side:       types.BUY,
		FeeRateBps: 0,
		Nonce:      0,
		Expiration: 0,
		Taker:      types.ZeroAddress,
	}

	// Order 2: Buy small amount of second highest volume market
	order2Args := &types.OrderArgs{
		TokenID:    tokenID2,
		Price:      0.04,  // Low price for testing
		Size:       25.0,  // $1 worth at 0.04
		Side:       types.BUY,
		FeeRateBps: 0,
		Nonce:      0,
		Expiration: 0,
		Taker:      types.ZeroAddress,
	}

	// Order 3: Buy small amount of third highest volume market
	order3Args := &types.OrderArgs{
		TokenID:    tokenID3,
		Price:      0.03,  // Low price for testing
		Size:       33.5,  // ~$1 worth at 0.03
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
	fmt.Printf("Order 1: BUY 20.00 %s @ $0.05 = $1.00\n", market1.Title)
	fmt.Printf("Order 2: BUY 25.00 %s @ $0.04 = $1.00\n", market2.Title)
	fmt.Printf("Order 3: BUY 33.50 %s @ $0.03 = $1.01\n", market3.Title)
	fmt.Println("Total value: $3.01")

	// Post the batch order
	fmt.Println("\nPosting batch order...")
	response, err := clobClient.PostOrders(batchOrders)
	if err != nil {
		fmt.Printf("Error posting batch order: %v\n", err)
		fmt.Println("\nCommon errors:")
		fmt.Println("- 'not enough balance/allowance': Account needs USDC balance")
		fmt.Println("- 'invalid amount': Each order size must be >= $1.00")
		fmt.Println("- 'maximum 5 orders': Batch is limited to 5 orders")
		fmt.Println("- '400 Invalid order payload': Check order format matches API spec")
		
		// Try to provide more specific error details
		if response != nil && response.ErrorMsg != "" {
			fmt.Printf("\nDetailed error: %s\n", response.ErrorMsg)
		}
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
				TokenID:    tokenID1,
				Price:      0.02,
				Size:       50.0, // $1 worth at 0.02
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
				TokenID:    tokenID2,
				Price:      0.02,
				Size:       50.0, // $1 worth at 0.02
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
