package main

import (
	"fmt"
	"log"
	
	"github.com/pooofdevelopment/go-clob-client/pkg/client"
)

// Example: Get markets
// Based on: py-clob-client-main/examples/get_markets.py
func main() {
	// Create a Level 0 client (no authentication needed)
	// Based on: py-clob-client-main/examples/get_markets.py:3-7
	host := "https://clob.polymarket.com"
	
	// Initialize client without auth
	clobClient, err := client.NewClobClient(host, 137, "", nil, nil, nil)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	
	// Get markets
	// Based on: py-clob-client-main/examples/get_markets.py:8
	response, err := clobClient.GetMarkets("")
	if err != nil {
		log.Fatal("Failed to get markets:", err)
	}
	
	// Check if we have data
	if data, ok := response["data"].([]interface{}); ok {
		fmt.Printf("Found %d markets\n", len(data))
		
		// Show first market if available
		if len(data) > 0 {
			if market, ok := data[0].(map[string]interface{}); ok {
				fmt.Printf("\nFirst market:\n")
				fmt.Printf("ID: %v\n", market["condition_id"])
				fmt.Printf("Question: %v\n", market["question"])
				fmt.Printf("Active: %v\n", market["active"])
			}
		}
	} else {
		fmt.Println("Response:", response)
	}
}