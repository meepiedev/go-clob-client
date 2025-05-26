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
- Proxy wallet support (POLY_PROXY signature type)

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

#### Method 1: Create new API credentials
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

#### Method 2: Use existing API credentials
```go
// If you already have API credentials
creds := &types.ApiCreds{
    ApiKey:        "your-api-key",
    ApiSecret:     "your-api-secret", 
    ApiPassphrase: "your-passphrase",
}

// Create L2 client directly
clobClient, err := client.NewClobClient(
    "https://clob.polymarket.com",
    137,
    "your_private_key_hex",
    creds,
    nil,
    nil,
)
```

### Proxy Wallet Support (POLY_PROXY)

The SDK supports proxy wallets for trading on behalf of another address:

```go
import "github.com/polymarket/go-order-utils/pkg/model"

// Create client with proxy wallet
sigType := model.POLY_PROXY
walletAddress := "0x8c1AEC5B133ACA324ff6d92083D8b7aBd552727e" // Your proxy wallet

clobClient, err := client.NewClobClient(
    "https://clob.polymarket.com",
    137,
    "your_private_key_hex",
    nil,
    &sigType,
    &walletAddress,
)

// Orders will be created with:
// - maker: proxy wallet (holds funds)
// - signer: your EOA address (from private key)
// - signatureType: 1 (POLY_PROXY)
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

See the `examples/` directory for complete working examples:

- `create_l2_client.go` - Complete L2 client setup with detailed explanations
- `create_and_post_order.go` - Create and post orders with proxy wallet support
- `get_ok.go` - Health check example
- `get_server_time.go` - Server timestamp retrieval
- `get_markets.go` - List active markets using Gamma API

## API Reference

The Go client maintains the same API structure as the Python client. Key methods include:

### Public Methods (L0)
- `GetOk()` - Health check
- `GetServerTime()` - Server timestamp
- `GetOrderBook(tokenID)` - Get orderbook
- `GetMidpoint(tokenID)` - Get mid-market price
- `GetPrice(tokenID, side)` - Get market price
- `GetMarkets(cursor)` - List markets
- `GetGammaMarkets(params)` - List markets from Gamma API

### Authenticated Methods (L1)
- `CreateOrder(args, options)` - Create signed order
- `CreateMarketOrder(args, options)` - Create market order
- `CreateApiKey(nonce)` - Create API key
- `DeriveApiKey(nonce)` - Derive existing API key

### Full Access Methods (L2)
- `PostOrder(order, type)` - Submit order
- `Cancel(orderID)` - Cancel order
- `CancelAll()` - Cancel all orders
- `GetOrders(params, cursor)` - Get user orders
- `GetTrades(params, cursor)` - Get trade history
- `GetBalanceAllowance(params)` - Get balances
- `GetNotifications(params, cursor)` - Get notifications
- `DropNotifications(params)` - Mark notifications as read

## Important Notes

### Order Size Requirements
- Minimum order size: $1.00 for most markets
- Orders below minimum will be rejected with "invalid amount" error

### Address Format
- Ensure wallet addresses are exactly 40 hex characters (excluding "0x" prefix)
- The SDK will handle address checksumming automatically

### Signature Types
- EOA (0): Standard Ethereum account signing
- POLY_PROXY (1): Proxy wallet signing (maker != signer)

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

## Troubleshooting

### "Invalid signature" errors
- Ensure your private key matches the expected signer address
- For POLY_PROXY, verify the proxy wallet relationship is established
- Check that addresses are properly formatted (40 hex chars)

### "Not enough balance/allowance" errors
- Verify your account has sufficient USDC balance
- Check token allowances are set for the exchange contract

## License

This project is licensed under the same terms as the original Python CLOB client.

## Acknowledgments

This is a Go port of Polymarket's [py-clob-client](https://github.com/Polymarket/py-clob-client), maintaining full compatibility with the original Python implementation.