package client

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/polymarket/go-order-utils/pkg/model"
	"github.com/pooofdevelopment/go-clob-client/pkg/config"
	"github.com/pooofdevelopment/go-clob-client/pkg/errors"
	"github.com/pooofdevelopment/go-clob-client/pkg/headers"
	"github.com/pooofdevelopment/go-clob-client/pkg/httpclient"
	"github.com/pooofdevelopment/go-clob-client/pkg/orderbuilder"
	"github.com/pooofdevelopment/go-clob-client/pkg/signer"
	"github.com/pooofdevelopment/go-clob-client/pkg/types"
	"github.com/pooofdevelopment/go-clob-client/pkg/utilities"
	"github.com/pooofdevelopment/go-clob-client/pkg/websocket"
)

// ClobClient is the main client for interacting with the CLOB API
// Based on: py-clob-client-main/py_clob_client/client.py:89-127
type ClobClient struct {
	host       string
	chainID    int
	signer     *signer.Signer
	creds      *types.ApiCreds
	mode       int
	builder    *orderbuilder.OrderBuilder
	httpClient *httpclient.Client

	// Local cache
	// Based on: py-clob-client-main/py_clob_client/client.py:123-124
	tickSizes map[string]types.TickSize
	negRisk   map[string]bool
}

// NewClobClient creates a new CLOB client
// Based on: py-clob-client-main/py_clob_client/client.py:90-127
func NewClobClient(host string, chainID int, privateKey string, creds *types.ApiCreds, signatureType *model.SignatureType, funder *string) (*ClobClient, error) {
	// Normalize host URL
	// Based on: py-clob-client-main/py_clob_client/client.py:111
	if strings.HasSuffix(host, "/") {
		host = host[:len(host)-1]
	}

	// Create signer if private key provided
	// Based on: py-clob-client-main/py_clob_client/client.py:113
	var s *signer.Signer
	var err error
	if privateKey != "" {
		s, err = signer.NewSigner(privateKey, chainID)
		if err != nil {
			return nil, fmt.Errorf("failed to create signer: %w", err)
		}
	}

	client := &ClobClient{
		host:       host,
		chainID:    chainID,
		signer:     s,
		creds:      creds,
		httpClient: httpclient.NewClient(),
		tickSizes:  make(map[string]types.TickSize),
		negRisk:    make(map[string]bool),
	}

	// Set client mode
	// Based on: py-clob-client-main/py_clob_client/client.py:115
	client.mode = client.getClientMode()

	// Create order builder if signer is available
	// Based on: py-clob-client-main/py_clob_client/client.py:117-120
	if s != nil {
		// Normalize funder address to lowercase if provided
		var normalizedFunder *string
		if funder != nil && *funder != "" {
			lower := strings.ToLower(*funder)
			normalizedFunder = &lower
		}
		client.builder = orderbuilder.NewOrderBuilder(s, signatureType, normalizedFunder)
	}

	return client, nil
}

// ClientOption is a functional option for configuring the ClobClient
type ClientOption func(*ClobClient)

// WithHTTPClient returns a ClientOption that sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *ClobClient) {
		c.httpClient = httpclient.NewClientWithHTTPClient(httpClient)
	}
}

// NewClobClientWithOptions creates a new CLOB client with custom options
func NewClobClientWithOptions(host string, chainID int, privateKey string, creds *types.ApiCreds, signatureType *model.SignatureType, funder *string, opts ...ClientOption) (*ClobClient, error) {
	// Create the client using the standard constructor
	client, err := NewClobClient(host, chainID, privateKey, creds, signatureType, funder)
	if err != nil {
		return nil, err
	}
	
	// Apply options
	for _, opt := range opts {
		opt(client)
	}
	
	return client, nil
}

// GetAddress returns the public address of the signer
// Based on: py-clob-client-main/py_clob_client/client.py:128-132
func (c *ClobClient) GetAddress() string {
	if c.signer != nil {
		return c.signer.Address()
	}
	return ""
}

// GetCollateralAddress returns the collateral token address
// Based on: py-clob-client-main/py_clob_client/client.py:134-141
func (c *ClobClient) GetCollateralAddress() (string, error) {
	contractConfig, err := config.GetContractConfig(c.chainID, false)
	if err != nil {
		return "", err
	}
	return contractConfig.Collateral, nil
}

// GetConditionalAddress returns the conditional token address
// Based on: py-clob-client-main/py_clob_client/client.py:143-148
func (c *ClobClient) GetConditionalAddress() (string, error) {
	contractConfig, err := config.GetContractConfig(c.chainID, false)
	if err != nil {
		return "", err
	}
	return contractConfig.ConditionalTokens, nil
}

// GetExchangeAddress returns the exchange address
// Based on: py-clob-client-main/py_clob_client/client.py:150-156
func (c *ClobClient) GetExchangeAddress(negRisk bool) (string, error) {
	contractConfig, err := config.GetContractConfig(c.chainID, negRisk)
	if err != nil {
		return "", err
	}
	return contractConfig.Exchange, nil
}

// GetOk performs a health check
// Based on: py-clob-client-main/py_clob_client/client.py:158-163
func (c *ClobClient) GetOk() (map[string]interface{}, error) {
	return c.httpClient.Get(c.host+"/", nil)
}

// GetServerTime returns the current server timestamp
// Based on: py-clob-client-main/py_clob_client/client.py:165-170
func (c *ClobClient) GetServerTime() (map[string]interface{}, error) {
	return c.httpClient.Get(c.host+types.TIME, nil)
}

// CreateApiKey creates a new CLOB API key
// Based on: py-clob-client-main/py_clob_client/client.py:172-191
func (c *ClobClient) CreateApiKey(nonce *int) (*types.ApiCreds, error) {
	if err := c.assertLevel1Auth(); err != nil {
		return nil, err
	}

	endpoint := c.host + types.CREATE_API_KEY
	headers, err := headers.CreateLevel1Headers(c.signer, nonce)
	if err != nil {
		return nil, err
	}

	response, err := c.httpClient.Post(endpoint, headers, nil)
	if err != nil {
		return nil, err
	}

	// Parse response
	// Based on: py-clob-client-main/py_clob_client/client.py:182-191
	creds := &types.ApiCreds{}
	if apiKey, ok := response["apiKey"].(string); ok {
		creds.ApiKey = apiKey
	}
	if secret, ok := response["secret"].(string); ok {
		creds.ApiSecret = secret
	}
	if passphrase, ok := response["passphrase"].(string); ok {
		creds.ApiPassphrase = passphrase
	}

	return creds, nil
}

// DeriveApiKey derives an existing CLOB API key
// Based on: py-clob-client-main/py_clob_client/client.py:193-212
func (c *ClobClient) DeriveApiKey(nonce *int) (*types.ApiCreds, error) {
	if err := c.assertLevel1Auth(); err != nil {
		return nil, err
	}

	endpoint := c.host + types.DERIVE_API_KEY
	headers, err := headers.CreateLevel1Headers(c.signer, nonce)
	if err != nil {
		return nil, err
	}

	response, err := c.httpClient.Get(endpoint, headers)
	if err != nil {
		return nil, err
	}

	// Parse response
	// Based on: py-clob-client-main/py_clob_client/client.py:204-212
	creds := &types.ApiCreds{}
	if apiKey, ok := response["apiKey"].(string); ok {
		creds.ApiKey = apiKey
	}
	if secret, ok := response["secret"].(string); ok {
		creds.ApiSecret = secret
	}
	if passphrase, ok := response["passphrase"].(string); ok {
		creds.ApiPassphrase = passphrase
	}

	return creds, nil
}

// CreateOrDeriveApiCreds creates API creds if not already created, otherwise derives them
// Based on: py-clob-client-main/py_clob_client/client.py:214-221
func (c *ClobClient) CreateOrDeriveApiCreds(nonce *int) (*types.ApiCreds, error) {
	// Try to create first
	creds, err := c.CreateApiKey(nonce)
	if err == nil {
		return creds, nil
	}

	// If creation fails, try to derive
	return c.DeriveApiKey(nonce)
}

// SetApiCreds sets the client API credentials
// Based on: py-clob-client-main/py_clob_client/client.py:223-228
func (c *ClobClient) SetApiCreds(creds *types.ApiCreds) {
	c.creds = creds
	c.mode = c.getClientMode()
}

// SetHTTPClient sets a custom HTTP client for the ClobClient
func (c *ClobClient) SetHTTPClient(httpClient *http.Client) {
	c.httpClient.SetHTTPClient(httpClient)
}

// GetApiKeys gets the available API keys for this address
// Based on: py-clob-client-main/py_clob_client/client.py:230-239
func (c *ClobClient) GetApiKeys() (map[string]interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	requestArgs := &types.RequestArgs{
		Method:      "GET",
		RequestPath: types.GET_API_KEYS,
	}

	h, err := headers.CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	return c.httpClient.Get(c.host+types.GET_API_KEYS, h)
}

// GetMidpoint gets the mid market price for the given market
// Based on: py-clob-client-main/py_clob_client/client.py:263-267
func (c *ClobClient) GetMidpoint(tokenID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s%s?token_id=%s", c.host, types.MID_POINT, tokenID)
	return c.httpClient.Get(url, nil)
}

// GetMidpoints gets the mid market prices for a set of token ids
// Based on: py-clob-client-main/py_clob_client/client.py:269-274
func (c *ClobClient) GetMidpoints(params []types.BookParams) (map[string]interface{}, error) {
	body := make([]map[string]string, len(params))
	for i, param := range params {
		body[i] = map[string]string{"token_id": param.TokenID}
	}
	return c.httpClient.Post(c.host+types.MID_POINTS, nil, body)
}

// GetPrice gets the market price for the given market and side
// Based on: py-clob-client-main/py_clob_client/client.py:276-280
func (c *ClobClient) GetPrice(tokenID string, side string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s%s?token_id=%s&side=%s", c.host, types.PRICE, tokenID, side)
	return c.httpClient.Get(url, nil)
}

// GetTickSize gets the tick size for a market
// Based on: py-clob-client-main/py_clob_client/client.py:302-309
func (c *ClobClient) GetTickSize(tokenID string) (types.TickSize, error) {
	// Check cache first
	// Based on: py-clob-client-main/py_clob_client/client.py:303-304
	if tickSize, ok := c.tickSizes[tokenID]; ok {
		return tickSize, nil
	}

	url := fmt.Sprintf("%s%s?token_id=%s", c.host, types.GET_TICK_SIZE, tokenID)
	result, err := c.httpClient.Get(url, nil)
	if err != nil {
		return "", err
	}

	// Parse and cache result
	// Based on: py-clob-client-main/py_clob_client/client.py:307
	// Handle both string and float64 responses
	var tickSizeStr string
	
	switch v := result["minimum_tick_size"].(type) {
	case string:
		tickSizeStr = v
	case float64:
		tickSizeStr = fmt.Sprintf("%g", v)
	default:
		// Try alternative field name
		switch v := result["min_tick_size"].(type) {
		case string:
			tickSizeStr = v
		case float64:
			tickSizeStr = fmt.Sprintf("%g", v)
		default:
			return "", fmt.Errorf("failed to get tick size from response: %v", result)
		}
	}
	
	tickSize := types.TickSize(tickSizeStr)
	c.tickSizes[tokenID] = tickSize
	return tickSize, nil
}

// GetNegRisk checks if a market uses neg risk
// Based on: py-clob-client-main/py_clob_client/client.py:311-318
func (c *ClobClient) GetNegRisk(tokenID string) (bool, error) {
	// Check cache first
	// Based on: py-clob-client-main/py_clob_client/client.py:312-313
	if negRisk, ok := c.negRisk[tokenID]; ok {
		return negRisk, nil
	}

	url := fmt.Sprintf("%s%s?token_id=%s", c.host, types.GET_NEG_RISK, tokenID)
	result, err := c.httpClient.Get(url, nil)
	if err != nil {
		return false, err
	}

	// Parse and cache result
	// Based on: py-clob-client-main/py_clob_client/client.py:316
	if negRisk, ok := result["neg_risk"].(bool); ok {
		c.negRisk[tokenID] = negRisk
		return negRisk, nil
	}

	return false, fmt.Errorf("failed to get neg risk")
}

// resolveTickSize resolves the tick size for an order
// Based on: py-clob-client-main/py_clob_client/client.py:320-334
func (c *ClobClient) resolveTickSize(tokenID string, tickSize *types.TickSize) (types.TickSize, error) {
	minTickSize, err := c.GetTickSize(tokenID)
	if err != nil {
		return "", err
	}

	if tickSize != nil {
		// Validate provided tick size
		// Based on: py-clob-client-main/py_clob_client/client.py:324-332
		if utilities.IsTickSizeSmaller(string(*tickSize), string(minTickSize)) {
			return "", errors.NewInvalidTickSizeError(string(*tickSize), string(minTickSize))
		}
		return *tickSize, nil
	}

	return minTickSize, nil
}

// CreateOrder creates and signs an order
// Based on: py-clob-client-main/py_clob_client/client.py:336-373
func (c *ClobClient) CreateOrder(orderArgs *types.OrderArgs, options *types.PartialCreateOrderOptions) (*model.SignedOrder, error) {
	if err := c.assertLevel1Auth(); err != nil {
		return nil, err
	}

	// Resolve tick size
	// Based on: py-clob-client-main/py_clob_client/client.py:345-350
	var tickSizePtr *types.TickSize
	if options != nil {
		tickSizePtr = options.TickSize
	}
	tickSize, err := c.resolveTickSize(orderArgs.TokenID, tickSizePtr)
	if err != nil {
		return nil, err
	}

	// Validate price
	// Based on: py-clob-client-main/py_clob_client/client.py:352-359
	if !utilities.PriceValid(orderArgs.Price, string(tickSize)) {
		maxPrice := 1 - float64(tickSize[0]-'0')/10
		return nil, errors.NewInvalidPriceError(orderArgs.Price, string(tickSize), fmt.Sprintf("%.4f", maxPrice))
	}

	// Get neg risk flag
	// Based on: py-clob-client-main/py_clob_client/client.py:361-365
	negRisk := false
	if options != nil && options.NegRisk != nil {
		negRisk = *options.NegRisk
	} else {
		negRisk, _ = c.GetNegRisk(orderArgs.TokenID)
	}

	// Create order
	// Based on: py-clob-client-main/py_clob_client/client.py:367-373
	createOptions := &types.CreateOrderOptions{
		TickSize: tickSize,
		NegRisk:  negRisk,
	}

	return c.builder.CreateOrder(orderArgs, createOptions)
}

// assertLevel1Auth checks for Level 1 authentication
// Based on: py-clob-client-main/py_clob_client/client.py:584-589
func (c *ClobClient) assertLevel1Auth() error {
	if c.mode < types.L1 {
		return errors.ErrL1AuthUnavailable
	}
	return nil
}

// assertLevel2Auth checks for Level 2 authentication
// Based on: py-clob-client-main/py_clob_client/client.py:591-596
func (c *ClobClient) assertLevel2Auth() error {
	if c.mode < types.L2 {
		return errors.ErrL2AuthUnavailable
	}
	return nil
}

// getClientMode determines the client authentication mode
// Based on: py-clob-client-main/py_clob_client/client.py:598-603
func (c *ClobClient) getClientMode() int {
	if c.signer != nil && c.creds != nil {
		return types.L2
	}
	if c.signer != nil {
		return types.L1
	}
	return types.L0
}

// CalculateMarketPrice calculates the matching price considering an amount and the current orderbook
// Based on: py-clob-client-main/py_clob_client/client.py:733-747
func (c *ClobClient) CalculateMarketPrice(tokenID string, side string, amount float64) (float64, error) {
	book, err := c.GetOrderBook(tokenID)
	if err != nil {
		return 0, err
	}

	if side == types.BUY {
		if len(book.Asks) == 0 {
			return 0, errors.ErrNoMatch
		}
		return c.builder.CalculateBuyMarketPrice(book.Asks, amount)
	} else {
		if len(book.Bids) == 0 {
			return 0, errors.ErrNoMatch
		}
		return c.builder.CalculateSellMarketPrice(book.Bids, amount)
	}
}

// GetOrderBook fetches the orderbook for the token_id
// Based on: py-clob-client-main/py_clob_client/client.py:518-523
func (c *ClobClient) GetOrderBook(tokenID string) (*types.OrderBookSummary, error) {
	url := fmt.Sprintf("%s%s?token_id=%s", c.host, types.GET_ORDER_BOOK, tokenID)
	rawObs, err := c.httpClient.Get(url, nil)
	if err != nil {
		return nil, err
	}

	return utilities.ParseRawOrderbookSummary(rawObs)
}

// CreateWebSocketClient creates a new websocket client for real-time data
// Based on: clob-client-main/examples/socketConnection.ts
func (c *ClobClient) CreateWebSocketClient(handler websocket.MessageHandler) *websocket.Client {
	// Use the official Polymarket CLOB websocket endpoint
	// Based on: https://docs.polymarket.com/developers/CLOB/websocket/wss-overview
	wsHost := "wss://ws-subscriptions-clob.polymarket.com"
	
	return websocket.NewClient(wsHost, handler)
}

// SubscribeToMarketData creates a websocket connection and subscribes to market data
// Based on: clob-client-main/examples/socketConnection.ts:63
func (c *ClobClient) SubscribeToMarketData(tokenIDs []string, handler websocket.MessageHandler) (*websocket.Client, error) {
	client := c.CreateWebSocketClient(handler)
	
	if err := client.SubscribeToMarket(tokenIDs, true); err != nil {
		_ = client.Close() // Best effort cleanup
		return nil, fmt.Errorf("failed to subscribe to market: %w", err)
	}
	
	return client, nil
}

// SubscribeToUserData creates a websocket connection and subscribes to user data
// Based on: clob-client-main/examples/socketConnection.ts:61
func (c *ClobClient) SubscribeToUserData(markets []string, handler websocket.MessageHandler) (*websocket.Client, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}
	
	client := c.CreateWebSocketClient(handler)
	
	if err := client.SubscribeToUser(c.creds, markets, true); err != nil {
		_ = client.Close() // Best effort cleanup
		return nil, fmt.Errorf("failed to subscribe to user data: %w", err)
	}
	
	return client, nil
}
