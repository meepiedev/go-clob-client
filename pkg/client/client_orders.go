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

// CreateAndPostOrder utility function to create and publish an order
// Based on: py-clob-client-main/py_clob_client/client.py:434-441
func (c *ClobClient) CreateAndPostOrder(orderArgs *types.OrderArgs, options *types.PartialCreateOrderOptions) (map[string]interface{}, error) {
	order, err := c.CreateOrder(orderArgs, options)
	if err != nil {
		return nil, err
	}
	
	return c.PostOrder(order, types.OrderTypeGTC)
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