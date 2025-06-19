package main

import (
	"fmt"
	"log"

	"github.com/pooofdevelopment/go-clob-client/pkg/client"
)

func main() {
	// Initialize the client
	c, err := client.NewClobClient("https://clob.polymarket.com", 137, "", nil, nil, nil)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	fmt.Println("\n\n=== Fetching All Negative Risk Events ===")

	eventsResponse, err := c.GetNegRiskEvents()
	if err != nil {
		log.Fatal("Failed to get negative risk events:", err)
	}

	if events, ok := eventsResponse["data"].([]interface{}); ok {
		fmt.Printf("Total negative risk events fetched: %d\n", len(events))

		// Show first 3 events as example
		for i := 0; i < 3 && i < len(events); i++ {
			if event, ok := events[i].(map[string]interface{}); ok {
				fmt.Printf("\nEvent %d:\n", i+1)
				fmt.Printf("  Slug: %v\n", event["slug"])
				fmt.Printf("  Title: %v\n", event["title"])
				fmt.Printf("  Volume: %v\n", event["volume"])
				fmt.Printf("  NegRisk: %v\n", event["negRisk"])
				fmt.Printf("  Active: %v\n", event["active"])
				fmt.Printf("  Archived: %v\n", event["archived"])
				if desc, ok := event["description"]; ok {
					// Show first 100 chars of description
					descStr := fmt.Sprintf("%v", desc)
					if len(descStr) > 100 {
						descStr = descStr[:100] + "..."
					}
					fmt.Printf("  Description: %s\n", descStr)
				}
			}
		}
	}
}
