# Polymarket Arbitrage Bot

This example demonstrates a negative risk arbitrage bot for Polymarket using the go-clob-sdk.

## Features

- **Terminal UI**: Real-time display of all orderbooks with color-coded status
- **Automatic Token ID Fetching**: Simply provide a market slug and the bot automatically fetches all outcome token IDs
- **Dual Strategy Support**: 
  - Taker strategy: Immediately buy all "No" shares when opportunity detected
  - Maker-taker strategy: Place limit orders and sweep remaining when one fills
- **Performance Monitoring**: Tracks capture-to-order latency and execution times
- **Concurrent Execution**: Uses goroutines for maximum speed

## Configuration

The bot uses a JSON configuration file with the following structure:

```json
{
  "private_key": "your_private_key",
  "address": "your_proxy_wallet_address",
  "signature_type": "poly_proxy",
  "markets": [
    {
      "name": "PGA TOUR Memorial Tournament Winner",
      "slug": "pga-tour-memorial-tournament-winner"
    }
  ],
  "strategies": {
    "taker": {
      "enabled": true,
      "min_edge": 0.01,
      "worker_amount": 2,
      "poll_interval_ms": 100
    },
    "maker-taker": {
      "enabled": false,
      "min_edge": 0.005,
      "extra_edge": 0.01,
      "worker_amount": 1,
      "poll_interval_ms": 200
    }
  }
}
```

## Usage

```bash
go run arbitrage_bot.go arbitrage_config.json
```

## How It Works

### Negative Risk Arbitrage

For a market with n mutually exclusive outcomes, if the sum of all "No" share prices is less than n-1, there's an arbitrage opportunity. The bot continuously monitors markets for these opportunities.

### Automatic Token ID Fetching

When you provide a market slug (e.g., "pga-tour-memorial-tournament-winner"), the bot:
1. Fetches all related markets from the Polymarket API
2. Extracts the "No" outcome token IDs for each player
3. Monitors all these outcomes for arbitrage opportunities

### Performance

The bot is optimized for low latency with:
- Concurrent market data updates
- Parallel order execution
- Sub-microsecond capture-to-order latency (typically ~281ns)

## Terminal UI

The bot displays a real-time terminal interface showing:

- **Market Overview**: All outcomes with their current best ask prices
- **Color Coding**:
  - Red: No market data available
  - Yellow: Active limit order placed
  - Green: Normal market data
  - Blinking Green: Arbitrage opportunity detected!
- **Summary Statistics**: Total cost, arbitrage edge, and performance metrics
- **Live Updates**: Orderbooks update every 100ms

## Example Output

```
Polymarket Arbitrage Bot - Live Monitor

Runtime: 45s | Opportunities: 2 | Orders: 108/110 | Last Update: 14:32:05

Outcome                        Best Ask        Ask Size        Total Cost      Arbitrage Edge      Status
─────────────────────────────────────────────────────────────────────────────────────────────────────

▶ PGA TOUR Memorial Tournament Winner (n=54)
  61953799645560191853...     $0.0234         45.00
  42872425868744231710...     $0.0156         122.50
  20417361785154968661...     $0.0445         89.00
  ...
  TOTAL                                                       $52.8500        0.1500 (0.28%)      OPPORTUNITY!

Press 'q' or ESC to quit | Logs: arbitrage_debug.log
```