package main

import (
	"fmt"
	"log"

	"github.com/pooofdevelopment/go-clob-client/pkg/client"
)

// Example: Get server time
// Based on: py-clob-client-main/examples/get_server_time.py
func main() {
	// Create a Level 0 client (no authentication needed)
	// Based on: py-clob-client-main/examples/get_server_time.py:3-7
	host := "https://clob.polymarket.com"

	// Initialize client without auth
	clobClient, err := client.NewClobClient(host, 137, "", nil, nil, nil)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	// Get server time
	// Based on: py-clob-client-main/examples/get_server_time.py:8
	response, err := clobClient.GetServerTime()
	if err != nil {
		log.Fatal("Failed to get server time:", err)
	}

	fmt.Println("Server time:", response)
}
