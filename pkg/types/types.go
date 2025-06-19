package types

import (
	"math/big"
	"time"
)

// ApiCreds represents API credentials for Level 2 authentication
// Based on: py-clob-client-main/py_clob_client/clob_types.py:10-14
type ApiCreds struct {
	ApiKey        string `json:"api_key"`
	ApiSecret     string `json:"api_secret"`
	ApiPassphrase string `json:"api_passphrase"`
}

// RequestArgs represents arguments for building request headers
// Based on: py-clob-client-main/py_clob_client/clob_types.py:17-21
type RequestArgs struct {
	Method      string      `json:"method"`
	RequestPath string      `json:"request_path"`
	Body        interface{} `json:"body,omitempty"`
}

// BookParams represents parameters for orderbook queries
// Based on: py-clob-client-main/py_clob_client/clob_types.py:24-27
type BookParams struct {
	TokenID string `json:"token_id"`
	Side    string `json:"side,omitempty"`
}

// OrderArgs represents arguments for creating an order
// Based on: py-clob-client-main/py_clob_client/clob_types.py:30-70
type OrderArgs struct {
	TokenID    string  `json:"token_id"`     // TokenID of the Conditional token asset being traded
	Price      float64 `json:"price"`        // Price used to create the order
	Size       float64 `json:"size"`         // Size in terms of the ConditionalToken
	Side       string  `json:"side"`         // Side of the order (BUY/SELL)
	FeeRateBps int     `json:"fee_rate_bps"` // Fee rate, in basis points, charged to the order maker
	Nonce      int     `json:"nonce"`        // Nonce used for onchain cancellations
	Expiration int64   `json:"expiration"`   // Timestamp after which the order is expired
	Taker      string  `json:"taker"`        // Address of the order taker. Zero address for public order
}

// MarketOrderArgs represents arguments for creating a market order
// Based on: py-clob-client-main/py_clob_client/clob_types.py:73-109
type MarketOrderArgs struct {
	TokenID    string  `json:"token_id"` // TokenID of the Conditional token asset being traded
	Amount     float64 `json:"amount"`   // BUY orders: $$$ Amount to buy, SELL orders: Shares to sell
	Side       string  `json:"side"`     // Side of the order
	Price      float64 `json:"price"`    // Price used to create the order (optional, calculated if 0)
	FeeRateBps int     `json:"fee_rate_bps"`
	Nonce      int     `json:"nonce"`
	Taker      string  `json:"taker"`
}

// TradeParams represents parameters for querying trades
// Based on: py-clob-client-main/py_clob_client/clob_types.py:112-119
type TradeParams struct {
	ID           string `json:"id,omitempty"`
	MakerAddress string `json:"maker_address,omitempty"`
	Market       string `json:"market,omitempty"`
	AssetID      string `json:"asset_id,omitempty"`
	Before       int64  `json:"before,omitempty"`
	After        int64  `json:"after,omitempty"`
}

// OpenOrderParams represents parameters for querying open orders
// Based on: py-clob-client-main/py_clob_client/clob_types.py:122-126
type OpenOrderParams struct {
	ID      string `json:"id,omitempty"`
	Market  string `json:"market,omitempty"`
	AssetID string `json:"asset_id,omitempty"`
}

// DropNotificationParams represents parameters for dropping notifications
// Based on: py-clob-client-main/py_clob_client/clob_types.py:129-131
type DropNotificationParams struct {
	IDs []string `json:"ids,omitempty"`
}

// OrderSummary represents a single order in the orderbook
// Based on: py-clob-client-main/py_clob_client/clob_types.py:134-145
type OrderSummary struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

// OrderBookSummary represents the full orderbook state
// Based on: py-clob-client-main/py_clob_client/clob_types.py:148-163
type OrderBookSummary struct {
	Market    string         `json:"market"`
	AssetID   string         `json:"asset_id"`
	Timestamp string         `json:"timestamp"`
	Bids      []OrderSummary `json:"bids"`
	Asks      []OrderSummary `json:"asks"`
	Hash      string         `json:"hash,omitempty"`
}

// AssetType represents the type of asset (collateral or conditional)
// Based on: py-clob-client-main/py_clob_client/clob_types.py:166-168
type AssetType string

const (
	AssetTypeCollateral  AssetType = "COLLATERAL"
	AssetTypeConditional AssetType = "CONDITIONAL"
)

// BalanceAllowanceParams represents parameters for balance/allowance queries
// Based on: py-clob-client-main/py_clob_client/clob_types.py:171-175
type BalanceAllowanceParams struct {
	AssetType     AssetType `json:"asset_type,omitempty"`
	TokenID       string    `json:"token_id,omitempty"`
	SignatureType int       `json:"signature_type"`
}

// OrderType represents the order type
// Based on: py-clob-client-main/py_clob_client/clob_types.py:178-182
type OrderType string

const (
	OrderTypeGTC OrderType = "GTC" // Good Till Cancelled
	OrderTypeFOK OrderType = "FOK" // Fill Or Kill
	OrderTypeGTD OrderType = "GTD" // Good Till Date
	OrderTypeFAK OrderType = "FAK" // Fill And Kill
)

// OrderScoringParams represents parameters for checking if order is scoring
// Based on: py-clob-client-main/py_clob_client/clob_types.py:185-187
type OrderScoringParams struct {
	OrderID string `json:"orderId"`
}

// OrdersScoringParams represents parameters for checking if multiple orders are scoring
// Based on: py-clob-client-main/py_clob_client/clob_types.py:190-192
type OrdersScoringParams struct {
	OrderIDs []string `json:"orderIds"`
}

// TickSize represents valid tick sizes for orders
// Based on: py-clob-client-main/py_clob_client/clob_types.py:195
type TickSize string

const (
	TickSize01    TickSize = "0.1"
	TickSize001   TickSize = "0.01"
	TickSize0001  TickSize = "0.001"
	TickSize00001 TickSize = "0.0001"
)

// CreateOrderOptions represents options for creating an order
// Based on: py-clob-client-main/py_clob_client/clob_types.py:198-201
type CreateOrderOptions struct {
	TickSize TickSize `json:"tick_size"`
	NegRisk  bool     `json:"neg_risk"`
}

// PartialCreateOrderOptions represents optional order creation options
// Based on: py-clob-client-main/py_clob_client/clob_types.py:204-207
type PartialCreateOrderOptions struct {
	TickSize *TickSize `json:"tick_size,omitempty"`
	NegRisk  *bool     `json:"neg_risk,omitempty"`
}

// RoundConfig represents rounding configuration for different tick sizes
// Based on: py-clob-client-main/py_clob_client/clob_types.py:210-214
type RoundConfig struct {
	Price  int `json:"price"`
	Size   int `json:"size"`
	Amount int `json:"amount"`
}

// ContractConfig represents smart contract addresses
// Based on: py-clob-client-main/py_clob_client/clob_types.py:217-236
// Also references: go-order-utils-main/pkg/config/config.go:9-17
type ContractConfig struct {
	Exchange          string `json:"exchange"`           // The exchange contract responsible for matching orders
	Collateral        string `json:"collateral"`         // The ERC20 token used as collateral
	ConditionalTokens string `json:"conditional_tokens"` // The ERC1155 conditional tokens contract
}

// Response represents a generic API response with pagination
// Not directly in Python client but inferred from usage patterns
type Response struct {
	NextCursor string      `json:"next_cursor,omitempty"`
	Data       interface{} `json:"data,omitempty"`
}

// Trade represents a trade/fill
// Inferred from Python client API usage in py-clob-client-main/py_clob_client/client.py:550-569
type Trade struct {
	ID           string    `json:"id"`
	TradedAt     time.Time `json:"traded_at"`
	MakerOrderID string    `json:"maker_order_id"`
	TakerOrderID string    `json:"taker_order_id"`
	Side         string    `json:"side"`
	Size         string    `json:"size"`
	Price        string    `json:"price"`
	FeeRateBps   string    `json:"fee_rate_bps"`
	MakerAddress string    `json:"maker_address"`
	Market       string    `json:"market"`
	Outcome      string    `json:"outcome"`
	BucketIndex  int       `json:"bucket_index"`
	AssetID      string    `json:"asset_id"`
}

// Order represents an order
// Inferred from Python client API usage in py-clob-client-main/py_clob_client/client.py:497-516
type Order struct {
	ID           string    `json:"id"`
	OrderID      string    `json:"order_id"`
	Market       string    `json:"market"`
	Side         string    `json:"side"`
	OriginalSize string    `json:"original_size"`
	SizeMatched  string    `json:"size_matched"`
	Size         string    `json:"size"`
	Price        string    `json:"price"`
	State        string    `json:"state"`
	AssetID      string    `json:"asset_id"`
	MakerAddress string    `json:"maker_address"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
	Outcome      string    `json:"outcome"`
}

// Market represents a market
// Inferred from Python client API usage in py-clob-client-main/py_clob_client/client.py:707-731
type Market struct {
	ID              string                 `json:"id"`
	Question        string                 `json:"question"`
	Description     string                 `json:"description"`
	ConditionID     string                 `json:"condition_id"`
	Tokens          []MarketToken          `json:"tokens"`
	MinTickSize     string                 `json:"min_tick_size"`
	Active          bool                   `json:"active"`
	Closed          bool                   `json:"closed"`
	QuestionID      string                 `json:"question_id,omitempty"`
	MarketType      string                 `json:"market_type"`
	MarketSlug      string                 `json:"market_slug"`
	EndDateISO      *time.Time             `json:"end_date_iso,omitempty"`
	GameStartTime   *time.Time             `json:"game_start_time,omitempty"`
	AcceptingOrders bool                   `json:"accepting_orders"`
	Tags            []string               `json:"tags,omitempty"`
	NegRisk         bool                   `json:"neg_risk"`
	EnableOrderBook bool                   `json:"enable_order_book"`
	ActivePre       bool                   `json:"active_pre"`
	ClosedPre       bool                   `json:"closed_pre"`
	Rewards         map[string]interface{} `json:"rewards,omitempty"`
}

// MarketToken represents a token in a market
// Inferred from Market structure
type MarketToken struct {
	TokenID string  `json:"token_id"`
	Outcome string  `json:"outcome"`
	Price   float64 `json:"price"`
	Winner  bool    `json:"winner"`
}

// Notification represents a user notification
// Inferred from Python client API usage in py-clob-client-main/py_clob_client/client.py:605-617
type Notification struct {
	ID               string                 `json:"id"`
	Type             string                 `json:"type"`
	Data             map[string]interface{} `json:"data"`
	UserAddress      string                 `json:"user_address"`
	TimestampCreated time.Time              `json:"timestamp_created"`
	Read             bool                   `json:"read"`
}

// BalanceAllowance represents balance and allowance amounts
// Inferred from Python client API usage in py-clob-client-main/py_clob_client/client.py:631-659
type BalanceAllowance struct {
	Balance   *big.Int `json:"balance"`
	Allowance *big.Int `json:"allowance"`
}

// SignedOrder represents a signed order (from go-order-utils)
// Based on: go-order-utils-main/pkg/model/order.go:91-96
type SignedOrder struct {
	Order     interface{} `json:"order"`
	Signature []byte      `json:"signature"`
}

// PostOrdersArgs represents a single order in a batch order request
// Used for batch order creation via POST /orders endpoint
type PostOrdersArgs struct {
	Order     interface{} `json:"order"`     // The signed order object
	OrderType OrderType   `json:"orderType"` // Order type (FOK, GTC, GTD, FAK)
}

// BatchOrderResponse represents the response from posting batch orders
// Based on the batch order API documentation
type BatchOrderResponse struct {
	Success     bool     `json:"success"`     // Whether the request was successful
	ErrorMsg    string   `json:"errorMsg"`    // Error message if unsuccessful
	OrderID     string   `json:"orderId"`     // ID of the order (for single order responses)
	OrderHashes []string `json:"orderHashes"` // Hashes of settlement transactions for marketable orders
}

// GammaMarketsParams represents parameters for gamma markets API
type GammaMarketsParams struct {
	Limit           int      `json:"limit,omitempty"`
	Offset          int      `json:"offset,omitempty"`
	Order           string   `json:"order,omitempty"`
	Ascending       *bool    `json:"ascending,omitempty"`
	ID              []int    `json:"id,omitempty"`
	Slug            []string `json:"slug,omitempty"`
	Archived        *bool    `json:"archived,omitempty"`
	Active          *bool    `json:"active,omitempty"`
	Closed          *bool    `json:"closed,omitempty"`
	Restricted      *bool    `json:"restricted,omitempty"`
	ClobTokenIDs    []string `json:"clob_token_ids,omitempty"`
	ConditionIDs    []string `json:"condition_ids,omitempty"`
	LiquidityNumMin float64  `json:"liquidity_num_min,omitempty"`
	LiquidityNumMax float64  `json:"liquidity_num_max,omitempty"`
	VolumeNumMin    float64  `json:"volume_num_min,omitempty"`
	VolumeNumMax    float64  `json:"volume_num_max,omitempty"`
	StartDateMin    string   `json:"start_date_min,omitempty"`
	StartDateMax    string   `json:"start_date_max,omitempty"`
	EndDateMin      string   `json:"end_date_min,omitempty"`
	EndDateMax      string   `json:"end_date_max,omitempty"`
	TagID           int      `json:"tag_id,omitempty"`
	RelatedTags     *bool    `json:"related_tags,omitempty"`
}

// GammaMarket represents a market from the gamma API
type GammaMarket struct {
	ID              int      `json:"id"`
	Slug            string   `json:"slug"`
	Archived        bool     `json:"archived"`
	Active          bool     `json:"active"`
	Closed          bool     `json:"closed"`
	Liquidity       float64  `json:"liquidity"`
	Volume          float64  `json:"volume"`
	StartDate       string   `json:"start_date"`
	EndDate         string   `json:"end_date"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	ConditionID     string   `json:"condition_id"`
	ClobTokenIDs    []string `json:"clob_token_ids"`
	EnableOrderBook bool     `json:"enable_order_book"`
	Question        string   `json:"question"`
	OrderMinSize    float64  `json:"orderMinSize"`
}

// GammaEventsParams represents parameters for gamma events API
type GammaEventsParams struct {
	ID   int    `json:"id,omitempty"`
	Slug string `json:"slug,omitempty"`
}

// GammaEvent represents an event from the gamma API
type GammaEvent struct {
	ID      int           `json:"id"`
	Slug    string        `json:"slug"`
	Title   string        `json:"title"`
	Markets []GammaMarket `json:"markets"`
	NegRisk bool          `json:"negRisk"`
}

// MarketsParams represents parameters for fetching markets
type MarketsParams struct {
	Active   bool `json:"active,omitempty"`   // Filter for active markets only
	Archived bool `json:"archived,omitempty"` // Filter for archived markets
	Limit    int  `json:"limit,omitempty"`    // Results per page (max 100)
	Offset   int  `json:"offset,omitempty"`   // Pagination offset
}

// MarketResponse represents a single market from the API
type MarketResponse struct {
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	NegRisk bool   `json:"neg_risk"`
	Volume  string `json:"volume"` // May be string or float in API
	Active  bool   `json:"active"`
	// Add other relevant fields as needed
}
