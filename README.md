# Go CLOB Client

A Go client library for interacting with Polymarket's CLOB (Central Limit Order Book) API. This is a complete port of the Python CLOB client to Go, maintaining feature parity and API compatibility.

## Features

- Complete implementation of all Python client functionality
- Three authentication levels:
  - L0: No authentication (public endpoints only)
  - L1: Private key authentication
  - L2: API key authentication (full access)
- Order creation and management
- Market data access
- Orderbook queries
- Trade history
- Balance and allowance management
- Notification handling

## Installation

```bash
go get github.com/pooofdevelopment/go-clob-client
```

## Quick Start

### Level 0 Client (No Authentication)
```go
package main

import (
    "fmt"
    "log"
    
    "github.com/pooofdevelopment/go-clob-client/pkg/client"
)

func main() {
    // Create client
    clobClient, err := client.NewClobClient("https://clob.polymarket.com", 137, "", nil, nil, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    // Get orderbook
    orderbook, err := clobClient.GetOrderBook("token_id")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(orderbook)
}
```

### Level 1 Client (Private Key Authentication)
```go
// Create client with private key
clobClient, err := client.NewClobClient(
    "https://clob.polymarket.com",
    137,
    "your_private_key_hex",
    nil,
    nil,
    nil,
)
```

### Level 2 Client (Full Authentication)
```go
// Create Level 1 client first
clobClient, err := client.NewClobClient(
    "https://clob.polymarket.com",
    137,
    "your_private_key_hex",
    nil,
    nil,
    nil,
)

// Create or derive API credentials
creds, err := clobClient.CreateOrDeriveApiCreds(nil)
if err != nil {
    log.Fatal(err)
}

// Upgrade to Level 2
clobClient.SetApiCreds(creds)
```

## Creating and Posting Orders

```go
orderArgs := &types.OrderArgs{
    TokenID:    "your_token_id",
    Price:      0.5,
    Size:       100,
    Side:       types.BUY,
    FeeRateBps: 0,
    Nonce:      0,
    Expiration: 0,
    Taker:      types.ZeroAddress,
}

// Create signed order
signedOrder, err := clobClient.CreateOrder(orderArgs, nil)
if err != nil {
    log.Fatal(err)
}

// Post order
response, err := clobClient.PostOrder(signedOrder, types.OrderTypeGTC)
if err != nil {
    log.Fatal(err)
}
```

## Examples

See the `examples/` directory for more detailed examples covering:
- Health checks
- Order creation
- Market orders
- Orderbook queries
- Trade history
- And more...

## API Reference

The Go client maintains the same API structure as the Python client. Key methods include:

### Public Methods (L0)
- `GetOk()` - Health check
- `GetServerTime()` - Server timestamp
- `GetOrderBook(tokenID)` - Get orderbook
- `GetMidpoint(tokenID)` - Get mid-market price
- `GetPrice(tokenID, side)` - Get market price
- `GetMarkets(cursor)` - List markets

### Authenticated Methods (L1)
- `CreateOrder(args, options)` - Create signed order
- `CreateMarketOrder(args, options)` - Create market order
- `CreateApiKey(nonce)` - Create API key
- `DeriveApiKey(nonce)` - Derive existing API key

### Full Access Methods (L2)
- `PostOrder(order, type)` - Submit order
- `Cancel(orderID)` - Cancel order
- `GetOrders(params, cursor)` - Get user orders
- `GetTrades(params, cursor)` - Get trade history
- `GetBalanceAllowance(params)` - Get balances

## Development

### Building
```bash
make build
```

### Testing
```bash
make test
```

### Linting
```bash
make lint
```

## License

This project is licensed under the same terms as the original Python CLOB client.

## Acknowledgments

This is a Go port of Polymarket's [py-clob-client](https://github.com/Polymarket/py-clob-client), maintaining full compatibility with the original Python implementation.