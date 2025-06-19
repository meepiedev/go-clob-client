package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pooofdevelopment/go-clob-client/pkg/types"
	"github.com/pooofdevelopment/go-clob-client/pkg/utilities"
)

// SubscriptionType represents the type of websocket subscription
// Based on: clob-client-main/examples/socketConnection.ts:29
type SubscriptionType string

const (
	SubscriptionTypeMarket SubscriptionType = "market"
	SubscriptionTypeUser   SubscriptionType = "user"
)

// SubscriptionMessage represents the subscription message sent to the websocket
// Based on: clob-client-main/examples/socketConnection.ts:16-23
type SubscriptionMessage struct {
	Auth        *AuthMessage `json:"auth,omitempty"`
	Type        string       `json:"type"`
	Markets     []string     `json:"markets"`
	AssetsIDs   []string     `json:"assets_ids"`
	InitialDump bool         `json:"initial_dump,omitempty"`
}

// AuthMessage represents authentication credentials for user subscriptions
// Based on: clob-client-main/examples/socketConnection.ts:18
type AuthMessage struct {
	APIKey     string `json:"apiKey"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}

// OrderBookUpdate represents a real-time orderbook update
// Based on: Polymarket CLOB WebSocket API documentation
type OrderBookUpdate struct {
	EventType   string                 `json:"event_type"`
	AssetID     string                 `json:"asset_id"`
	Market      string                 `json:"market"`
	MarketSlug  string                 `json:"market_slug,omitempty"`
	Slug        string                 `json:"slug,omitempty"`
	Timestamp   string                 `json:"timestamp"`
	Hash        string                 `json:"hash,omitempty"`
	// API sometimes uses buys/sells, sometimes bids/asks
	Buys        []types.OrderSummary   `json:"buys,omitempty"`
	Sells       []types.OrderSummary   `json:"sells,omitempty"`
	Bids        []types.OrderSummary   `json:"bids,omitempty"`
	Asks        []types.OrderSummary   `json:"asks,omitempty"`
	// For backwards compatibility
	Data        types.OrderBookSummary `json:"data,omitempty"`
}

// PriceChangeUpdate represents a price change event
// Based on: Polymarket CLOB WebSocket API documentation
type PriceChangeUpdate struct {
	EventType string            `json:"event_type"` // "price_change"
	AssetID   string            `json:"asset_id"`
	Market    string            `json:"market"`
	Timestamp string            `json:"timestamp"`
	Hash      string            `json:"hash"`
	Changes   []PriceChange     `json:"changes"`
}

// PriceChange represents an individual price level change
// Based on: Polymarket CLOB WebSocket API documentation
type PriceChange struct {
	Price string `json:"price"`
	Side  string `json:"side"` // "BUY" or "SELL"
	Size  string `json:"size"`
}

// TickSizeChangeUpdate represents a tick size change event
// Based on: Polymarket CLOB WebSocket API documentation
type TickSizeChangeUpdate struct {
	EventType   string `json:"event_type"` // "tick_size_change"
	AssetID     string `json:"asset_id"`
	Market      string `json:"market"`
	Timestamp   string `json:"timestamp"`
	OldTickSize string `json:"old_tick_size"`
	NewTickSize string `json:"new_tick_size"`
}

// LastTradePriceUpdate represents a last trade price event
// Based on: Polymarket CLOB WebSocket API behavior
type LastTradePriceUpdate struct {
	EventType string `json:"event_type"` // "last_trade_price"
	AssetID   string `json:"asset_id"`
	Market    string `json:"market"`
	Timestamp string `json:"timestamp"`
	Price     string `json:"price"`
	Size      string `json:"size,omitempty"`
}

// UserUpdate represents a real-time user update (orders, trades)
// Inferred from websocket API behavior for user channel
type UserUpdate struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp int64                  `json:"timestamp"`
}

// MessageHandler defines the interface for handling websocket messages
type MessageHandler interface {
	OnOrderBookUpdate(update *OrderBookUpdate)
	OnPriceChange(update *PriceChangeUpdate)
	OnTickSizeChange(update *TickSizeChangeUpdate)
	OnLastTradePrice(update *LastTradePriceUpdate)
	OnUserUpdate(update *UserUpdate)
	OnError(err error)
	OnConnect()
	OnDisconnect()
}

// Client represents a websocket client for Polymarket CLOB
// Based on: clob-client-main/examples/socketConnection.ts
type Client struct {
	host             string
	conn             *websocket.Conn
	handler          MessageHandler
	ctx              context.Context
	cancel           context.CancelFunc
	mu               sync.RWMutex
	isConnected      bool
	pingTicker       *time.Ticker
	lastOrderbookHash string // Track last orderbook to avoid duplicate empty updates
}

// NewClient creates a new websocket client
func NewClient(host string, handler MessageHandler) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		host:    host,
		handler: handler,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Connect establishes a websocket connection to a specific channel
// Based on: clob-client-main/examples/socketConnection.ts:32
func (c *Client) connectToChannel(channel string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Parse the URL and convert to websocket scheme
	u, err := url.Parse(c.host)
	if err != nil {
		return fmt.Errorf("invalid host URL: %w", err)
	}

	// Convert HTTP/HTTPS to WS/WSS
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
		// Already correct
	default:
		return fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
	}

	// Append the channel path
	u.Path = fmt.Sprintf("/ws/%s", channel)
	
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to websocket: %w", err)
	}

	// Close existing connection if any
	if c.conn != nil {
		c.conn.Close()
	}

	c.conn = conn
	c.isConnected = true

	// Start the message handler goroutine
	go c.messageLoop()

	// Start heartbeat
	// Based on: clob-client-main/examples/socketConnection.ts:83-86
	c.startHeartbeat()

	if c.handler != nil {
		c.handler.OnConnect()
	}

	return nil
}

// Connect establishes a websocket connection (deprecated - use specific subscription methods)
func (c *Client) Connect() error {
	return fmt.Errorf("use SubscribeToMarket or SubscribeToUser instead of Connect")
}

// SubscribeToMarket subscribes to market data for specific token IDs
// Based on: clob-client-main/examples/socketConnection.ts:62-64
func (c *Client) SubscribeToMarket(tokenIDs []string, initialDump bool) error {
	// Connect to market channel (no auth required)
	if err := c.connectToChannel("market"); err != nil {
		return fmt.Errorf("failed to connect to market channel: %w", err)
	}

	return c.subscribe(SubscriptionTypeMarket, nil, nil, tokenIDs, initialDump)
}

// SubscribeToUser subscribes to user data for specific markets
// Based on: clob-client-main/examples/socketConnection.ts:60-61
func (c *Client) SubscribeToUser(creds *types.ApiCreds, markets []string, initialDump bool) error {
	if creds == nil {
		return fmt.Errorf("credentials required for user subscription")
	}

	// Connect to user channel (auth required)
	if err := c.connectToChannel("user"); err != nil {
		return fmt.Errorf("failed to connect to user channel: %w", err)
	}

	// Based on: clob-client-main/examples/socketConnection.ts:46-53
	auth := &AuthMessage{
		APIKey:     creds.ApiKey,
		Secret:     creds.ApiSecret,
		Passphrase: creds.ApiPassphrase,
	}

	return c.subscribe(SubscriptionTypeUser, auth, markets, nil, initialDump)
}

// subscribe sends a subscription message to the websocket
// Based on: clob-client-main/examples/socketConnection.ts:45-64,75-81
func (c *Client) subscribe(subType SubscriptionType, auth *AuthMessage, markets []string, assetIDs []string, initialDump bool) error {
	c.mu.RLock()
	if !c.isConnected || c.conn == nil {
		c.mu.RUnlock()
		return fmt.Errorf("client is not connected")
	}
	conn := c.conn
	c.mu.RUnlock()

	// Create subscription message
	// Based on: clob-client-main/examples/socketConnection.ts:45-64
	subMsg := SubscriptionMessage{
		Auth:        auth,
		Type:        string(subType),
		Markets:     markets,
		AssetsIDs:   assetIDs,
		InitialDump: initialDump,
	}

	if markets == nil {
		subMsg.Markets = []string{}
	}
	if assetIDs == nil {
		subMsg.AssetsIDs = []string{}
	}

	// Send subscription message
	// Based on: clob-client-main/examples/socketConnection.ts:75-81
	msgBytes, err := json.Marshal(subMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription message: %w", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, msgBytes)
	if err != nil {
		return fmt.Errorf("failed to send subscription message: %w", err)
	}

	log.Printf("Subscribed to %s with message: %s", subType, string(msgBytes))
	return nil
}

// startHeartbeat starts the ping heartbeat
// Based on: clob-client-main/examples/socketConnection.ts:83-86
func (c *Client) startHeartbeat() {
	c.pingTicker = time.NewTicker(30 * time.Second)
	go func() {
		for {
			select {
			case <-c.ctx.Done():
				return
			case <-c.pingTicker.C:
				c.mu.RLock()
				if c.isConnected && c.conn != nil {
					conn := c.conn
					c.mu.RUnlock()
					// Based on: clob-client-main/examples/socketConnection.ts:85
					err := conn.WriteMessage(websocket.TextMessage, []byte("PING"))
					if err != nil {
						log.Printf("Failed to send ping: %v", err)
						if c.handler != nil {
							c.handler.OnError(err)
						}
					} else {
						log.Printf("[WS PING] Sent PING to WebSocket")
					}
				} else {
					c.mu.RUnlock()
				}
			}
		}
	}()
}

// messageLoop handles incoming websocket messages
// Based on: clob-client-main/examples/socketConnection.ts:93-95
func (c *Client) messageLoop() {
	defer func() {
		c.mu.Lock()
		c.isConnected = false
		if c.conn != nil {
			c.conn.Close()
		}
		c.mu.Unlock()

		if c.handler != nil {
			c.handler.OnDisconnect()
		}
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				return
			}

			_, message, err := conn.ReadMessage()
			if err != nil {
				if c.handler != nil {
					c.handler.OnError(fmt.Errorf("websocket read error: %w", err))
				}
				return
			}

			// Handle PONG responses
			if string(message) == "PONG" {
				log.Printf("[WS PONG] Received PONG from WebSocket")
				continue
			}

			// Try to parse as different message formats
			// First try as an array (common for orderbook data)
			var arrayUpdate []interface{}
			if err := json.Unmarshal(message, &arrayUpdate); err == nil {
				c.handleArrayMessage(arrayUpdate, message)
				continue
			}

			// Try to parse as object
			var objectUpdate map[string]interface{}
			if err := json.Unmarshal(message, &objectUpdate); err == nil {
				c.handleObjectMessage(objectUpdate, message)
				continue
			}

			log.Printf("Failed to parse message: %s", string(message))
		}
	}
}

// handleArrayMessage processes array-format websocket messages
// This is common for orderbook data from Polymarket
func (c *Client) handleArrayMessage(arrayData []interface{}, rawMessage []byte) {
	if c.handler == nil {
		return
	}

	// For orderbook data, the array contains orderbook objects
	for _, item := range arrayData {
		if itemMap, ok := item.(map[string]interface{}); ok {
			// Log the raw item for debugging
			itemBytes, err := json.Marshal(itemMap)
			if err != nil {
				log.Printf("Failed to marshal array item: %v", err)
				continue
			}
			
			// Check what type of event this is
			// if eventType, hasEventType := itemMap["event_type"].(string); hasEventType {
			// 	log.Printf("Received %s event", eventType) // Disabled for TUI
			// }
			
			// Log bid/ask counts for debugging
			// if bids, hasBids := itemMap["bids"].([]interface{}); hasBids {
			// 	if asks, hasAsks := itemMap["asks"].([]interface{}); hasAsks {
			// 		log.Printf("Message contains %d bids, %d asks", len(bids), len(asks)) // Disabled for TUI
			// 	}
			// }
			
			c.handleObjectMessage(itemMap, itemBytes)
		}
	}
}

// handleObjectMessage processes object-format websocket messages
// Based on: Polymarket CLOB WebSocket API documentation
func (c *Client) handleObjectMessage(update map[string]interface{}, rawMessage []byte) {
	if c.handler == nil {
		return
	}

	// Check for event_type field to determine message type
	// Based on: Polymarket CLOB WebSocket API documentation
	if eventType, hasEventType := update["event_type"].(string); hasEventType {
		switch eventType {
		case "book":
			// Parse as book message (full orderbook snapshot)
			var bookUpdate OrderBookUpdate
			if err := json.Unmarshal(rawMessage, &bookUpdate); err == nil {
				// Normalize between buys/sells and bids/asks formats
				if len(bookUpdate.Buys) == 0 && len(bookUpdate.Bids) > 0 {
					bookUpdate.Buys = bookUpdate.Bids
				}
				if len(bookUpdate.Sells) == 0 && len(bookUpdate.Asks) > 0 {
					bookUpdate.Sells = bookUpdate.Asks
				}
				
				// Debug: Log if we have a slug field (temporary for debugging)
				if bookUpdate.MarketSlug != "" || bookUpdate.Slug != "" {
					log.Printf("DEBUG: Found slug in orderbook: market_slug='%s' slug='%s'", bookUpdate.MarketSlug, bookUpdate.Slug)
				}
				
				// log.Printf("Parsed book message: %d buys, %d sells", len(bookUpdate.Buys), len(bookUpdate.Sells)) // Disabled for TUI
				c.handler.OnOrderBookUpdate(&bookUpdate)
				return
			} else {
				log.Printf("Failed to parse book message: %v", err)
				log.Printf("Raw book message: %s", string(rawMessage))
			}
			
		case "price_change":
			// Parse as price change message (incremental updates)
			var priceUpdate PriceChangeUpdate
			if err := json.Unmarshal(rawMessage, &priceUpdate); err == nil {
				c.handler.OnPriceChange(&priceUpdate)
				return
			} else {
				log.Printf("Failed to parse price_change message: %v", err)
			}
			
		case "tick_size_change":
			// Parse as tick size change message
			var tickUpdate TickSizeChangeUpdate
			if err := json.Unmarshal(rawMessage, &tickUpdate); err == nil {
				c.handler.OnTickSizeChange(&tickUpdate)
				return
			} else {
				log.Printf("Failed to parse tick_size_change message: %v", err)
			}
			
		case "last_trade_price":
			// Parse as last trade price message
			var tradeUpdate LastTradePriceUpdate
			if err := json.Unmarshal(rawMessage, &tradeUpdate); err == nil {
				c.handler.OnLastTradePrice(&tradeUpdate)
				return
			} else {
				log.Printf("Failed to parse last_trade_price message: %v", err)
			}
			
		default:
			log.Printf("Unknown event_type: %s", eventType)
		}
	}

	// Fallback: Check if this looks like legacy orderbook data
	// This maintains compatibility with the old format
	if _, hasMarket := update["market"].(string); hasMarket {
		if _, hasAssetID := update["asset_id"].(string); hasAssetID {
			// Try to parse using the existing utility for backwards compatibility
			obs, err := utilities.ParseRawOrderbookSummary(update)
			if err == nil {
				// Convert to new format
				bookUpdate := &OrderBookUpdate{
					EventType: "book",
					AssetID:   obs.AssetID,
					Market:    obs.Market,
					Timestamp: obs.Timestamp,
					Buys:      obs.Bids,  // Map bids to buys
					Sells:     obs.Asks,  // Map asks to sells
					Data:      *obs,      // Keep for backwards compatibility
				}
				c.handler.OnOrderBookUpdate(bookUpdate)
				return
			} else {
				log.Printf("Failed to parse legacy orderbook data: %v", err)
			}
		}
	}

	// If we can't determine the type, treat as user update
	var userUpdate UserUpdate
	if err := json.Unmarshal(rawMessage, &userUpdate); err == nil {
		c.handler.OnUserUpdate(&userUpdate)
	} else {
		log.Printf("Unknown message format: %s", string(rawMessage))
	}
}

// Close closes the websocket connection
func (c *Client) Close() error {
	c.cancel()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.pingTicker != nil {
		c.pingTicker.Stop()
	}

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.isConnected = false
		return err
	}

	return nil
}

// IsConnected returns whether the client is currently connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}