package main

import (
	"fmt"
	"log"
	"os"
	
	"github.com/pooofdevelopment/go-clob-client/pkg/client"
	"github.com/pooofdevelopment/go-clob-client/pkg/types"
)

// Example: Create and post an order
// Based on: py-clob-client-main/examples/order.py
func main() {
	// Configuration
	// Based on: py-clob-client-main/examples/order.py:3-12
	host := "https://clob.polymarket.com"
	key := os.Getenv("PK") // Private key from environment
	
	if key == "" {
		log.Fatal("Please set PK environment variable with your private key")
	}
	
	// Create a Level 1 client for order creation
	clobClient, err := client.NewClobClient(host, 137, key, nil, nil, nil)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	
	// Create API credentials if needed for Level 2
	// Based on: py-clob-client-main/examples/order.py:13-18
	creds, err := clobClient.CreateOrDeriveApiCreds(nil)
	if err != nil {
		log.Fatal("Failed to create/derive API creds:", err)
	}
	
	// Upgrade to Level 2
	clobClient.SetApiCreds(creds)
	
	// Order parameters
	// Based on: py-clob-client-main/examples/order.py:20-39
	tokenID := "1343197538147866997676796723584448962074666613334415804986823996107646701544"
	
	orderArgs := &types.OrderArgs{
		TokenID:    tokenID,
		Price:      0.5,
		Size:       100,
		Side:       types.BUY,
		FeeRateBps: 0,
		Nonce:      0,
		Expiration: 0, // No expiration
		Taker:      types.ZeroAddress,
	}
	
	// Create a signed order
	// Based on: py-clob-client-main/examples/order.py:41
	signedOrder, err := clobClient.CreateOrder(orderArgs, nil)
	if err != nil {
		log.Fatal("Failed to create order:", err)
	}
	
	fmt.Println("Created order:", signedOrder)
	
	// Post the order
	// Based on: py-clob-client-main/examples/order.py:44
	response, err := clobClient.PostOrder(signedOrder, types.OrderTypeGTC)
	if err != nil {
		log.Fatal("Failed to post order:", err)
	}
	
	fmt.Println("Posted order response:", response)
}