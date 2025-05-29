package main

import (
	"fmt"
	"log"

	"github.com/pooofdevelopment/go-clob-client/pkg/client"
)

// Example: Health check
// Based on: py-clob-client-main/examples/get_ok.py
func main() {
	// Create a Level 0 client (no authentication needed for health check)
	// Based on: py-clob-client-main/examples/get_ok.py:3-7
	host := "https://clob.polymarket.com"

	// Initialize client without auth
	clobClient, err := client.NewClobClient(host, 137, "", nil, nil, nil)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	// Perform health check
	// Based on: py-clob-client-main/examples/get_ok.py:8
	response, err := clobClient.GetOk()
	if err != nil {
		log.Fatal("Health check failed:", err)
	}

	fmt.Println("Health check response:", response)
}
