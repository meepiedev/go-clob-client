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

// Example: Create and post an order with proxy wallet support
// This demonstrates:
// - Creating a Level 2 client with proxy wallet (POLY_PROXY)
// - Finding an active market
// - Creating and signing an order
// - Posting the order to the exchange
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
	if walletAddress != "" {
		fmt.Printf("  Proxy wallet: %s\n", walletAddress)
	}

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
		Limit:  5,
	}

	markets, err := clobClient.GetGammaMarkets(gammaParams)
	if err != nil {
		log.Fatal("Failed to get gamma markets:", err)
	}

	if len(markets) == 0 {
		log.Fatal("No active markets found")
	}

	// Find a market with token IDs
	var tokenID string
	var marketQuestion string
	for _, market := range markets {
		if len(market.ClobTokenIDs) > 0 {
			tokenID = market.ClobTokenIDs[0]
			marketQuestion = market.Question
			break
		}
	}

	if tokenID == "" {
		log.Fatal("No market with token IDs found")
	}

	fmt.Printf("\nSelected market: %s\n", marketQuestion)
	fmt.Printf("Token ID: %s\n", tokenID)

	// Create an order
	// Note: Minimum order size is typically $1.00
	orderArgs := &types.OrderArgs{
		TokenID:    tokenID,
		Price:      0.50,     // 50 cents
		Size:       10.0,     // $5.00 worth (10 shares at $0.50)
		Side:       types.BUY,
		FeeRateBps: 0,        // 0 basis points fee
		Nonce:      0,
		Expiration: 0,        // No expiration
		Taker:      types.ZeroAddress,
	}

	fmt.Println("\nCreating order...")
	fmt.Printf("  Side: %s\n", orderArgs.Side)
	fmt.Printf("  Price: $%.2f\n", orderArgs.Price)
	fmt.Printf("  Size: %.2f shares\n", orderArgs.Size)
	fmt.Printf("  Total value: $%.2f\n", orderArgs.Price*orderArgs.Size)

	signedOrder, err := clobClient.CreateOrder(orderArgs, nil)
	if err != nil {
		log.Fatal("Failed to create order:", err)
	}

	fmt.Println("✓ Order created and signed")

	// Display order details
	fmt.Println("\nOrder details:")
	orderJSON, _ := json.MarshalIndent(map[string]interface{}{
		"salt":          signedOrder.Salt.String(),
		"maker":         signedOrder.Maker.Hex(),
		"signer":        signedOrder.Signer.Hex(),
		"taker":         signedOrder.Taker.Hex(),
		"tokenId":       signedOrder.TokenId.String(),
		"makerAmount":   signedOrder.MakerAmount.String(),
		"takerAmount":   signedOrder.TakerAmount.String(),
		"expiration":    signedOrder.Expiration.String(),
		"nonce":         signedOrder.Nonce.String(),
		"feeRateBps":    signedOrder.FeeRateBps.String(),
		"side":          signedOrder.Side.String(),
		"signatureType": signedOrder.SignatureType.String(),
		"signature":     fmt.Sprintf("0x%x", signedOrder.Signature),
	}, "", "  ")
	fmt.Println(string(orderJSON))

	// Post the order
	fmt.Println("\nPosting order...")
	response, err := clobClient.PostOrder(signedOrder, types.OrderTypeGTC)
	if err != nil {
		fmt.Printf("Error posting order: %v\n", err)
		fmt.Println("\nCommon errors:")
		fmt.Println("- 'not enough balance/allowance': Account needs USDC balance")
		fmt.Println("- 'invalid amount': Order size must be >= $1.00")
		fmt.Println("- 'invalid signature': Check wallet address format (40 hex chars)")
		return
	}

	fmt.Println("✓ Order posted successfully!")
	fmt.Printf("Response: %+v\n", response)
}