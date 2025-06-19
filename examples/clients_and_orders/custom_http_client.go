package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/pooofdevelopment/go-clob-client/pkg/client"
)

func main() {
	// Example 1: Create a custom HTTP client with specific IP binding
	// This is useful for IP rotation scenarios
	
	// Create a custom dialer that binds to a specific local IP
	localAddr, err := net.ResolveTCPAddr("tcp", "192.168.1.100:0") // Replace with your IP
	if err != nil {
		log.Printf("Warning: Could not resolve local address: %v", err)
		localAddr = nil // Fall back to default
	}
	
	dialer := &net.Dialer{
		LocalAddr: localAddr,
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	
	// Create a custom transport with the dialer
	transport := &http.Transport{
		DialContext:         dialer.DialContext,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	
	// Create a custom HTTP client
	customHTTPClient := &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second, // Longer timeout for slow connections
	}
	
	// Example 2: Using the setter method
	fmt.Println("Example 1: Using SetHTTPClient method")
	
	// Create a standard client
	clobClient, err := client.NewClobClient(
		"https://clob.polymarket.com",
		137, // Polygon mainnet
		"",  // No private key for this example
		nil, // No API credentials
		nil, // No signature type
		nil, // No funder address
	)
	if err != nil {
		log.Fatal(err)
	}
	
	// Set the custom HTTP client
	clobClient.SetHTTPClient(customHTTPClient)
	
	// Test the connection
	ok, err := clobClient.GetOk()
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Connection test with custom client: %v\n", ok)
	}
	
	// Example 3: Using the functional options pattern
	fmt.Println("\nExample 2: Using WithHTTPClient option")
	
	// Create another custom HTTP client with proxy support
	proxyTransport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		// Uncomment and configure if you have a proxy:
		// Proxy: http.ProxyURL(proxyURL),
	}
	
	proxyHTTPClient := &http.Client{
		Transport: proxyTransport,
		Timeout:   45 * time.Second,
	}
	
	// Create client with custom HTTP client using options
	clobClient2, err := client.NewClobClientWithOptions(
		"https://clob.polymarket.com",
		137, // Polygon mainnet
		"",  // No private key for this example
		nil, // No API credentials
		nil, // No signature type
		nil, // No funder address
		client.WithHTTPClient(proxyHTTPClient),
	)
	if err != nil {
		log.Fatal(err)
	}
	
	// Test the connection
	serverTime, err := clobClient2.GetServerTime()
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Server time with custom client: %v\n", serverTime)
	}
	
	// Example 4: Custom client with retry logic
	fmt.Println("\nExample 3: Custom client with retry logic")
	
	retryTransport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
	}
	
	// Simple retry client (in production, use a proper retry library)
	retryClient := &http.Client{
		Transport: retryTransport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Custom redirect policy
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	
	// Create client and set custom HTTP client
	clobClient3, err := client.NewClobClient(
		"https://clob.polymarket.com",
		137, // Polygon mainnet
		"",  // No private key for this example
		nil, // No API credentials
		nil, // No signature type
		nil, // No funder address
	)
	if err != nil {
		log.Fatal(err)
	}
	
	clobClient3.SetHTTPClient(retryClient)
	
	// Fetch some market data
	markets, err := clobClient3.GetMarkets("")
	if err != nil {
		log.Printf("Error fetching markets: %v", err)
	} else {
		fmt.Printf("Successfully fetched %d markets with custom client\n", len(markets))
	}
}