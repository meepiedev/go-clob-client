package client

import (
	"encoding/json"
	"fmt"
	
	"github.com/polymarket/go-order-utils/pkg/model"
	"github.com/pooofdevelopment/go-clob-client/pkg/errors"
	"github.com/pooofdevelopment/go-clob-client/pkg/headers"
	"github.com/pooofdevelopment/go-clob-client/pkg/httpclient"
	"github.com/pooofdevelopment/go-clob-client/pkg/types"
	"github.com/pooofdevelopment/go-clob-client/pkg/utilities"
)

// CreateMarketOrder creates and signs a market order
// Based on: py-clob-client-main/py_clob_client/client.py:375-419
func (c *ClobClient) CreateMarketOrder(orderArgs *types.MarketOrderArgs, options *types.PartialCreateOrderOptions) (*model.SignedOrder, error) {
	if err := c.assertLevel1Auth(); err != nil {
		return nil, err
	}
	
	// Resolve tick size
	// Based on: py-clob-client-main/py_clob_client/client.py:386-391
	var tickSizePtr *types.TickSize
	if options != nil {
		tickSizePtr = options.TickSize
	}
	tickSize, err := c.resolveTickSize(orderArgs.TokenID, tickSizePtr)
	if err != nil {
		return nil, err
	}
	
	// Calculate market price if not provided
	// Based on: py-clob-client-main/py_clob_client/client.py:393-396
	if orderArgs.Price <= 0 {
		price, err := c.CalculateMarketPrice(orderArgs.TokenID, orderArgs.Side, orderArgs.Amount)
		if err != nil {
			return nil, err
		}
		orderArgs.Price = price
	}
	
	// Validate price
	// Based on: py-clob-client-main/py_clob_client/client.py:398-405
	if !utilities.PriceValid(orderArgs.Price, string(tickSize)) {
		maxPrice := 1 - float64(tickSize[0]-'0')/10
		return nil, errors.NewInvalidPriceError(orderArgs.Price, string(tickSize), fmt.Sprintf("%.4f", maxPrice))
	}
	
	// Get neg risk flag
	// Based on: py-clob-client-main/py_clob_client/client.py:407-411
	negRisk := false
	if options != nil && options.NegRisk != nil {
		negRisk = *options.NegRisk
	} else {
		negRisk, _ = c.GetNegRisk(orderArgs.TokenID)
	}
	
	// Create market order
	// Based on: py-clob-client-main/py_clob_client/client.py:413-419
	createOptions := &types.CreateOrderOptions{
		TickSize: tickSize,
		NegRisk:  negRisk,
	}
	
	return c.builder.CreateMarketOrder(orderArgs, createOptions)
}

// PostOrder posts the order to the exchange
// Based on: py-clob-client-main/py_clob_client/client.py:421-432
func (c *ClobClient) PostOrder(order *model.SignedOrder, orderType types.OrderType) (map[string]interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}
	
	// Convert order to JSON format
	// Based on: py-clob-client-main/py_clob_client/client.py:426
	body := c.orderToJSON(order, orderType)
	
	requestArgs := &types.RequestArgs{
		Method:      "POST",
		RequestPath: types.POST_ORDER,
		Body:        body,
	}
	
	h, err := headers.CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}
	
	return c.httpClient.Post(c.host+types.POST_ORDER, h, body)
}

// PostOrders posts multiple orders to the exchange in a batch
// Based on the batch order API documentation
func (c *ClobClient) PostOrders(orders []types.PostOrdersArgs) (*types.BatchOrderResponse, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}
	
	// Validate batch size
	if len(orders) == 0 {
		return nil, fmt.Errorf("at least one order is required")
	}
	
	// Build the request body as an array of orders
	body := make([]map[string]interface{}, len(orders))
	for i, orderArgs := range orders {
		// Extract the signed order
		signedOrder, ok := orderArgs.Order.(*model.SignedOrder)
		if !ok {
			return nil, fmt.Errorf("order at index %d is not a SignedOrder", i)
		}
		
		// Convert side from int to string
		sideStr := "BUY"
		if signedOrder.Side.Int64() == 1 {
			sideStr = "SELL"
		}
		
		// Build order JSON matching the API spec exactly
		orderData := map[string]interface{}{
			"salt":          signedOrder.Salt.Int64(),  // Send as number
			"maker":         signedOrder.Maker.Hex(),
			"signer":        signedOrder.Signer.Hex(),
			"taker":         signedOrder.Taker.Hex(),
			"tokenId":       signedOrder.TokenId.String(),
			"makerAmount":   signedOrder.MakerAmount.String(),
			"takerAmount":   signedOrder.TakerAmount.String(),
			"expiration":    signedOrder.Expiration.String(),
			"nonce":         signedOrder.Nonce.String(),
			"feeRateBps":    signedOrder.FeeRateBps.String(),
			"side":          sideStr,
			"signatureType": signedOrder.SignatureType.Int64(),  // Send as number
			"signature":     "0x" + fmt.Sprintf("%x", signedOrder.Signature),
		}
		
		// Build order object matching TypeScript/Python format
		body[i] = map[string]interface{}{
			"order":     orderData,
			"owner":     c.creds.ApiKey,
			"orderType": string(orderArgs.OrderType),
		}
	}
	
	requestArgs := &types.RequestArgs{
		Method:      "POST",
		RequestPath: types.POST_ORDERS,
		Body:        body,
	}
	
	h, err := headers.CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}
	
	response, err := c.httpClient.Post(c.host+types.POST_ORDERS, h, body)
	if err != nil {
		return nil, err
	}
	
	// The response might be an array of order responses or a single response object
	// Try to handle both cases
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}
	
	// First try as array response (successful batch)
	var arrayResponse []map[string]interface{}
	if err := json.Unmarshal(responseJSON, &arrayResponse); err == nil && len(arrayResponse) > 0 {
		// Extract order IDs and hashes from successful responses
		orderHashes := make([]string, 0)
		for _, orderResp := range arrayResponse {
			if hash, ok := orderResp["transactionHash"].(string); ok && hash != "" {
				orderHashes = append(orderHashes, hash)
			}
		}
		
		return &types.BatchOrderResponse{
			Success:     true,
			OrderHashes: orderHashes,
		}, nil
	}
	
	// Try as single error response
	var batchResponse types.BatchOrderResponse
	if err := json.Unmarshal(responseJSON, &batchResponse); err != nil {
		// If we can't parse it as expected format, return a generic error
		return &types.BatchOrderResponse{
			Success:  false,
			ErrorMsg: fmt.Sprintf("Unexpected response format: %s", string(responseJSON)),
		}, nil
	}
	
	return &batchResponse, nil
}

// CreateAndPostOrder utility function to create and publish an order
// Based on: py-clob-client-main/py_clob_client/client.py:434-441
func (c *ClobClient) CreateAndPostOrder(orderArgs *types.OrderArgs, options *types.PartialCreateOrderOptions) (map[string]interface{}, error) {
	order, err := c.CreateOrder(orderArgs, options)
	if err != nil {
		return nil, err
	}
	
	return c.PostOrder(order, types.OrderTypeGTC)
}

// CreateAndPostOrders utility function to create and publish multiple orders in a batch
// This is a convenience method that creates orders and posts them together
func (c *ClobClient) CreateAndPostOrders(ordersList []struct {
	Args      *types.OrderArgs
	Options   *types.PartialCreateOrderOptions
	OrderType types.OrderType
}) (*types.BatchOrderResponse, error) {
	// Create all orders first
	postOrdersArgs := make([]types.PostOrdersArgs, len(ordersList))
	
	for i, orderData := range ordersList {
		// Create the order
		order, err := c.CreateOrder(orderData.Args, orderData.Options)
		if err != nil {
			return nil, fmt.Errorf("failed to create order %d: %w", i, err)
		}
		
		// Add to batch
		postOrdersArgs[i] = types.PostOrdersArgs{
			Order:     order,
			OrderType: orderData.OrderType,
		}
	}
	
	// Post all orders as a batch
	return c.PostOrders(postOrdersArgs)
}

// Cancel cancels an order
// Based on: py-clob-client-main/py_clob_client/client.py:443-453
func (c *ClobClient) Cancel(orderID string) (map[string]interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}
	
	body := map[string]string{"orderID": orderID}
	
	requestArgs := &types.RequestArgs{
		Method:      "DELETE",
		RequestPath: types.CANCEL,
		Body:        body,
	}
	
	h, err := headers.CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}
	
	return c.httpClient.Delete(c.host+types.CANCEL, h, body)
}

// CancelOrders cancels multiple orders
// Based on: py-clob-client-main/py_clob_client/client.py:455-469
func (c *ClobClient) CancelOrders(orderIDs []string) (map[string]interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}
	
	body := orderIDs
	
	requestArgs := &types.RequestArgs{
		Method:      "DELETE",
		RequestPath: types.CANCEL_ORDERS,
		Body:        body,
	}
	
	h, err := headers.CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}
	
	return c.httpClient.Delete(c.host+types.CANCEL_ORDERS, h, body)
}

// CancelAll cancels all available orders for the user
// Based on: py-clob-client-main/py_clob_client/client.py:471-479
func (c *ClobClient) CancelAll() (map[string]interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}
	
	requestArgs := &types.RequestArgs{
		Method:      "DELETE",
		RequestPath: types.CANCEL_ALL,
	}
	
	h, err := headers.CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}
	
	return c.httpClient.Delete(c.host+types.CANCEL_ALL, h, nil)
}

// CancelMarketOrders cancels market orders
// Based on: py-clob-client-main/py_clob_client/client.py:481-495
func (c *ClobClient) CancelMarketOrders(market string, assetID string) (map[string]interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}
	
	body := map[string]string{
		"market":   market,
		"asset_id": assetID,
	}
	
	requestArgs := &types.RequestArgs{
		Method:      "DELETE",
		RequestPath: types.CANCEL_MARKET_ORDERS,
		Body:        body,
	}
	
	h, err := headers.CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}
	
	return c.httpClient.Delete(c.host+types.CANCEL_MARKET_ORDERS, h, body)
}

// GetOrders gets orders for the API key
// Based on: py-clob-client-main/py_clob_client/client.py:497-516
func (c *ClobClient) GetOrders(params *types.OpenOrderParams, nextCursor string) ([]types.Order, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}
	
	requestArgs := &types.RequestArgs{
		Method:      "GET",
		RequestPath: types.ORDERS,
	}
	
	h, err := headers.CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}
	
	var results []types.Order
	cursor := nextCursor
	if cursor == "" {
		cursor = "MA=="
	}
	
	// Paginate through results
	// Based on: py-clob-client-main/py_clob_client/client.py:507-515
	for cursor != types.EndCursor {
		url := httpclient.AddQueryOpenOrdersParams(c.host+types.ORDERS, params, cursor)
		response, err := c.httpClient.Get(url, h)
		if err != nil {
			return nil, err
		}
		
		// Parse cursor
		if nextCursor, ok := response["next_cursor"].(string); ok {
			cursor = nextCursor
		} else {
			break
		}
		
		// Parse data
		if data, ok := response["data"].([]interface{}); ok {
			for _, item := range data {
				// Convert to Order type
				orderJSON, _ := json.Marshal(item)
				var order types.Order
				if err := json.Unmarshal(orderJSON, &order); err == nil {
					results = append(results, order)
				}
			}
		}
	}
	
	return results, nil
}

// GetOrder fetches the order corresponding to the order_id
// Based on: py-clob-client-main/py_clob_client/client.py:539-548
func (c *ClobClient) GetOrder(orderID string) (map[string]interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}
	
	endpoint := types.GET_ORDER + orderID
	requestArgs := &types.RequestArgs{
		Method:      "GET",
		RequestPath: endpoint,
	}
	
	h, err := headers.CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}
	
	return c.httpClient.Get(c.host+endpoint, h)
}

// GetTrades fetches the trade history for a user
// Based on: py-clob-client-main/py_clob_client/client.py:550-569
func (c *ClobClient) GetTrades(params *types.TradeParams, nextCursor string) ([]types.Trade, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}
	
	requestArgs := &types.RequestArgs{
		Method:      "GET",
		RequestPath: types.TRADES,
	}
	
	h, err := headers.CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}
	
	var results []types.Trade
	cursor := nextCursor
	if cursor == "" {
		cursor = "MA=="
	}
	
	// Paginate through results
	// Based on: py-clob-client-main/py_clob_client/client.py:560-568
	for cursor != types.EndCursor {
		url := httpclient.AddQueryTradeParams(c.host+types.TRADES, params, cursor)
		response, err := c.httpClient.Get(url, h)
		if err != nil {
			return nil, err
		}
		
		// Parse cursor
		if nextCursor, ok := response["next_cursor"].(string); ok {
			cursor = nextCursor
		} else {
			break
		}
		
		// Parse data
		if data, ok := response["data"].([]interface{}); ok {
			for _, item := range data {
				// Convert to Trade type
				tradeJSON, _ := json.Marshal(item)
				var trade types.Trade
				if err := json.Unmarshal(tradeJSON, &trade); err == nil {
					results = append(results, trade)
				}
			}
		}
	}
	
	return results, nil
}

// orderToJSON converts an order to JSON format for API submission
// Based on: py-clob-client-main/py_clob_client/utilities.py:35-65
func (c *ClobClient) orderToJSON(order *model.SignedOrder, orderType types.OrderType) map[string]interface{} {
	// Convert side from int to string
	sideStr := "BUY"
	if order.Side.Int64() == 1 {
		sideStr = "SELL"
	}
	
	// Build order JSON with signature inside the order object
	// Python SDK sends salt and signatureType as numbers, not strings
	orderData := map[string]interface{}{
		"salt":          order.Salt.Int64(),  // Send as number, not string
		"maker":         order.Maker.Hex(),   // Don't lowercase - Python doesn't
		"signer":        order.Signer.Hex(),  // Don't lowercase - Python doesn't
		"taker":         order.Taker.Hex(),   // Don't lowercase - Python doesn't
		"tokenId":       order.TokenId.String(),
		"makerAmount":   order.MakerAmount.String(),
		"takerAmount":   order.TakerAmount.String(),
		"expiration":    order.Expiration.String(),
		"nonce":         order.Nonce.String(),
		"feeRateBps":    order.FeeRateBps.String(),
		"side":          sideStr,
		"signatureType": order.SignatureType.Int64(),  // Send as number, not string
		"signature":     "0x" + fmt.Sprintf("%x", order.Signature),
	}
	
	return map[string]interface{}{
		"order":     orderData,
		"owner":     c.creds.ApiKey,
		"orderType": string(orderType),
	}
}