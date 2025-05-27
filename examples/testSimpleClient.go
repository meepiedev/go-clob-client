package main

import (
	"fmt"
	"log"

	"github.com/pooofdevelopment/go-clob-client/pkg/client"
	"github.com/pooofdevelopment/go-clob-client/pkg/types"
)

func printOrderBook(orderbook *types.OrderBookSummary) {
	fmt.Printf("Market: %s\n", orderbook.Market)
	fmt.Printf("Asset ID: %s\n", orderbook.AssetID)
	fmt.Printf("Timestamp: %s\n", orderbook.Timestamp)
	fmt.Println()

	fmt.Println("ASKS (Sell Orders):")
	fmt.Println("Price\t\tSize")
	fmt.Println("-----\t\t----")
	for i := len(orderbook.Asks) - 1; i >= 0; i-- {
		ask := orderbook.Asks[i]
		if ask.Price != "" && ask.Size != "" {
			fmt.Printf("%s\t\t%s\n", ask.Price, ask.Size)
		}
	}
	
	fmt.Println()
	fmt.Println("BIDS (Buy Orders):")
	fmt.Println("Price\t\tSize")
	fmt.Println("-----\t\t----")
	for _, bid := range orderbook.Bids {
		if bid.Price != "" && bid.Size != "" {
			fmt.Printf("%s\t\t%s\n", bid.Price, bid.Size)
		}
	}
	
	if len(orderbook.Bids) == 0 && len(orderbook.Asks) == 0 {
		fmt.Println("No orders in the orderbook")
	}
}

func main() {
	// Create client
	clobClient, err := client.NewClobClient("https://clob.polymarket.com", 137, "", nil, nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Get orderbook for a specific token
	fmt.Println("Fetching orderbook...")
	orderbook, err := clobClient.GetOrderBook("64857617821618792090309776061594999588607561964140319397152984325528949636614")
	if err != nil {
		log.Fatal(err)
	}

	printOrderBook(orderbook)
}
