package main

import (
	"encoding/json"
	"fmt"
	"os"
	
	"github.com/pooofdevelopment/go-clob-client/pkg/client"
	"github.com/pooofdevelopment/go-clob-client/pkg/types"
)

func main() {
	// Check for required environment variables
	privateKey := os.Getenv("PK")
	if privateKey == "" {
		fmt.Println("Please set PK environment variable with your private key")
		return
	}

	// Configuration
	host := "https://clob.polymarket.com"
	chainID := 137 // Polygon mainnet

	// Create client
	clobClient, err := client.NewClobClient(host, chainID, privateKey, nil, nil, nil)
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}

	// Create/derive API credentials
	creds, err := clobClient.CreateOrDeriveApiCreds(nil)
	if err != nil {
		fmt.Printf("Failed to create/derive API credentials: %v\n", err)
		return
	}

	// Upgrade to Level 2
	clobClient.SetApiCreds(creds)

	// Create a sample order
	orderArgs := &types.OrderArgs{
		TokenID:    "71321045679252212594626385532706912750332728571942532289631379312455583992563",
		Price:      0.50,
		Size:       10.0,
		Side:       types.BUY,
		FeeRateBps: 0,
		Nonce:      0,
		Expiration: 0,
		Taker:      types.ZeroAddress,
	}

	signedOrder, err := clobClient.CreateOrder(orderArgs, nil)
	if err != nil {
		fmt.Printf("Failed to create order: %v\n", err)
		return
	}

	// Manually build the JSON format
	signedOrderTyped := signedOrder
	
	sideStr := "BUY"
	if signedOrderTyped.Side.Int64() == 1 {
		sideStr = "SELL"
	}
	
	orderData := map[string]interface{}{
		"salt":          signedOrderTyped.Salt.Int64(),
		"maker":         signedOrderTyped.Maker.Hex(),
		"signer":        signedOrderTyped.Signer.Hex(),
		"taker":         signedOrderTyped.Taker.Hex(),
		"tokenId":       signedOrderTyped.TokenId.String(),
		"makerAmount":   signedOrderTyped.MakerAmount.String(),
		"takerAmount":   signedOrderTyped.TakerAmount.String(),
		"expiration":    signedOrderTyped.Expiration.String(),
		"nonce":         signedOrderTyped.Nonce.String(),
		"feeRateBps":    signedOrderTyped.FeeRateBps.String(),
		"side":          sideStr,
		"signatureType": signedOrderTyped.SignatureType.Int64(),
		"signature":     "0x" + fmt.Sprintf("%x", signedOrderTyped.Signature),
	}
	
	// Build the batch order format
	batchOrder := []map[string]interface{}{
		{
			"order":     orderData,
			"owner":     creds.ApiKey,
			"orderType": "GTC",
		},
	}
	
	// Pretty print the JSON
	jsonBytes, err := json.MarshalIndent(batchOrder, "", "  ")
	if err != nil {
		fmt.Printf("Failed to marshal JSON: %v\n", err)
		return
	}
	
	fmt.Println("JSON format that will be sent to API:")
	fmt.Println(string(jsonBytes))
}