package client

import (
	"encoding/json"
	"fmt"

	"github.com/pooofdevelopment/go-clob-client/pkg/headers"
	"github.com/pooofdevelopment/go-clob-client/pkg/httpclient"
	"github.com/pooofdevelopment/go-clob-client/pkg/types"
	"github.com/pooofdevelopment/go-clob-client/pkg/utilities"
)

// GetClosedOnlyMode gets the closed only mode flag for this address
// Based on: py-clob-client-main/py_clob_client/client.py:241-250
func (c *ClobClient) GetClosedOnlyMode() (map[string]interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	requestArgs := &types.RequestArgs{
		Method:      "GET",
		RequestPath: types.CLOSED_ONLY,
	}

	h, err := headers.CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	return c.httpClient.Get(c.host+types.CLOSED_ONLY, h)
}

// DeleteApiKey deletes an API key
// Based on: py-clob-client-main/py_clob_client/client.py:252-261
func (c *ClobClient) DeleteApiKey() (map[string]interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	requestArgs := &types.RequestArgs{
		Method:      "DELETE",
		RequestPath: types.DELETE_API_KEY,
	}

	h, err := headers.CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	return c.httpClient.Delete(c.host+types.DELETE_API_KEY, h, nil)
}

// GetPrices gets the market prices for a set of tokens
// Based on: py-clob-client-main/py_clob_client/client.py:282-287
func (c *ClobClient) GetPrices(params []types.BookParams) (map[string]interface{}, error) {
	body := make([]map[string]string, len(params))
	for i, param := range params {
		body[i] = map[string]string{
			"token_id": param.TokenID,
			"side":     param.Side,
		}
	}
	return c.httpClient.Post(c.host+types.GET_PRICES, nil, body)
}

// GetSpread gets the spread for the given market
// Based on: py-clob-client-main/py_clob_client/client.py:289-293
func (c *ClobClient) GetSpread(tokenID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s%s?token_id=%s", c.host, types.GET_SPREAD, tokenID)
	return c.httpClient.Get(url, nil)
}

// GetSpreads gets the spreads for a set of token ids
// Based on: py-clob-client-main/py_clob_client/client.py:295-300
func (c *ClobClient) GetSpreads(params []types.BookParams) (map[string]interface{}, error) {
	body := make([]map[string]string, len(params))
	for i, param := range params {
		body[i] = map[string]string{"token_id": param.TokenID}
	}
	return c.httpClient.Post(c.host+types.GET_SPREADS, nil, body)
}

// GetOrderBooks fetches the orderbook for a set of token ids
// Based on: py-clob-client-main/py_clob_client/client.py:525-531
func (c *ClobClient) GetOrderBooks(params []types.BookParams) ([]types.OrderBookSummary, error) {
	body := make([]map[string]string, len(params))
	for i, param := range params {
		body[i] = map[string]string{"token_id": param.TokenID}
	}

	response, err := c.httpClient.Post(c.host+types.GET_ORDER_BOOKS, nil, body)
	if err != nil {
		return nil, err
	}

	// Parse response as array
	var results []types.OrderBookSummary
	if data, ok := response["data"].([]interface{}); ok {
		for _, item := range data {
			if raw, ok := item.(map[string]interface{}); ok {
				obs, err := utilities.ParseRawOrderbookSummary(raw)
				if err == nil {
					results = append(results, *obs)
				}
			}
		}
	}

	return results, nil
}

// GetOrderBookHash calculates the hash for the given orderbook
// Based on: py-clob-client-main/py_clob_client/client.py:533-537
func (c *ClobClient) GetOrderBookHash(orderbook *types.OrderBookSummary) string {
	return utilities.GenerateOrderbookSummaryHash(orderbook)
}

// GetLastTradePrice fetches the last trade price for token_id
// Based on: py-clob-client-main/py_clob_client/client.py:571-575
func (c *ClobClient) GetLastTradePrice(tokenID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s%s?token_id=%s", c.host, types.GET_LAST_TRADE_PRICE, tokenID)
	return c.httpClient.Get(url, nil)
}

// GetLastTradesPrices fetches the last trades prices for a set of token ids
// Based on: py-clob-client-main/py_clob_client/client.py:577-582
func (c *ClobClient) GetLastTradesPrices(params []types.BookParams) (map[string]interface{}, error) {
	body := make([]map[string]string, len(params))
	for i, param := range params {
		body[i] = map[string]string{"token_id": param.TokenID}
	}
	return c.httpClient.Post(c.host+types.GET_LAST_TRADES_PRICES, nil, body)
}

// GetNotifications fetches the notifications for a user
// Based on: py-clob-client-main/py_clob_client/client.py:605-617
func (c *ClobClient) GetNotifications() (map[string]interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	requestArgs := &types.RequestArgs{
		Method:      "GET",
		RequestPath: types.GET_NOTIFICATIONS,
	}

	h, err := headers.CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s%s?signature_type=%d", c.host, types.GET_NOTIFICATIONS, c.builder.GetSignatureType())
	return c.httpClient.Get(url, h)
}

// DropNotifications drops the notifications for a user
// Based on: py-clob-client-main/py_clob_client/client.py:619-629
func (c *ClobClient) DropNotifications(params *types.DropNotificationParams) (map[string]interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	requestArgs := &types.RequestArgs{
		Method:      "DELETE",
		RequestPath: types.DROP_NOTIFICATIONS,
	}

	h, err := headers.CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	url := httpclient.DropNotificationsQueryParams(c.host+types.DROP_NOTIFICATIONS, params)
	return c.httpClient.Delete(url, h, nil)
}

// GetBalanceAllowance fetches the balance & allowance for a user
// Based on: py-clob-client-main/py_clob_client/client.py:631-644
func (c *ClobClient) GetBalanceAllowance(params *types.BalanceAllowanceParams) (map[string]interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	requestArgs := &types.RequestArgs{
		Method:      "GET",
		RequestPath: types.GET_BALANCE_ALLOWANCE,
	}

	h, err := headers.CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	// Set default signature type if not provided
	// Based on: py-clob-client-main/py_clob_client/client.py:639-640
	if params.SignatureType == -1 {
		params.SignatureType = c.builder.GetSignatureType()
	}

	url := httpclient.AddBalanceAllowanceParamsToURL(c.host+types.GET_BALANCE_ALLOWANCE, params)
	return c.httpClient.Get(url, h)
}

// UpdateBalanceAllowance updates the balance & allowance for a user
// Based on: py-clob-client-main/py_clob_client/client.py:646-659
func (c *ClobClient) UpdateBalanceAllowance(params *types.BalanceAllowanceParams) (map[string]interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	requestArgs := &types.RequestArgs{
		Method:      "GET",
		RequestPath: types.UPDATE_BALANCE_ALLOWANCE,
	}

	h, err := headers.CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	// Set default signature type if not provided
	// Based on: py-clob-client-main/py_clob_client/client.py:654-655
	if params.SignatureType == -1 {
		params.SignatureType = c.builder.GetSignatureType()
	}

	url := httpclient.AddBalanceAllowanceParamsToURL(c.host+types.UPDATE_BALANCE_ALLOWANCE, params)
	return c.httpClient.Get(url, h)
}

// IsOrderScoring checks if the order is currently scoring
// Based on: py-clob-client-main/py_clob_client/client.py:661-672
func (c *ClobClient) IsOrderScoring(params *types.OrderScoringParams) (map[string]interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	requestArgs := &types.RequestArgs{
		Method:      "GET",
		RequestPath: types.IS_ORDER_SCORING,
	}

	h, err := headers.CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	url := httpclient.AddOrderScoringParamsToURL(c.host+types.IS_ORDER_SCORING, params)
	return c.httpClient.Get(url, h)
}

// AreOrdersScoring checks if the orders are currently scoring
// Based on: py-clob-client-main/py_clob_client/client.py:674-687
func (c *ClobClient) AreOrdersScoring(params *types.OrdersScoringParams) (map[string]interface{}, error) {
	if err := c.assertLevel2Auth(); err != nil {
		return nil, err
	}

	body := params.OrderIDs
	requestArgs := &types.RequestArgs{
		Method:      "POST",
		RequestPath: types.ARE_ORDERS_SCORING,
		Body:        body,
	}

	h, err := headers.CreateLevel2Headers(c.signer, c.creds, requestArgs)
	if err != nil {
		return nil, err
	}

	return c.httpClient.Post(c.host+types.ARE_ORDERS_SCORING, h, body)
}

// GetSamplingMarkets gets the current sampling markets
// Based on: py-clob-client-main/py_clob_client/client.py:689-695
func (c *ClobClient) GetSamplingMarkets(nextCursor string) (map[string]interface{}, error) {
	if nextCursor == "" {
		nextCursor = "MA=="
	}
	url := fmt.Sprintf("%s%s?next_cursor=%s", c.host, types.GET_SAMPLING_MARKETS, nextCursor)
	return c.httpClient.Get(url, nil)
}

// GetSamplingSimplifiedMarkets gets the current sampling simplified markets
// Based on: py-clob-client-main/py_clob_client/client.py:697-705
func (c *ClobClient) GetSamplingSimplifiedMarkets(nextCursor string) (map[string]interface{}, error) {
	if nextCursor == "" {
		nextCursor = "MA=="
	}
	url := fmt.Sprintf("%s%s?next_cursor=%s", c.host, types.GET_SAMPLING_SIMPLIFIED_MARKETS, nextCursor)
	return c.httpClient.Get(url, nil)
}

// GetMarkets gets the current markets
// Based on: py-clob-client-main/py_clob_client/client.py:707-711
func (c *ClobClient) GetMarkets(nextCursor string) (map[string]interface{}, error) {
	if nextCursor == "" {
		nextCursor = "MA=="
	}
	url := fmt.Sprintf("%s%s?next_cursor=%s", c.host, types.GET_MARKETS, nextCursor)
	return c.httpClient.Get(url, nil)
}

// GetSimplifiedMarkets gets the current simplified markets
// Based on: py-clob-client-main/py_clob_client/client.py:713-719
func (c *ClobClient) GetSimplifiedMarkets(nextCursor string) (map[string]interface{}, error) {
	if nextCursor == "" {
		nextCursor = "MA=="
	}
	url := fmt.Sprintf("%s%s?next_cursor=%s", c.host, types.GET_SIMPLIFIED_MARKETS, nextCursor)
	return c.httpClient.Get(url, nil)
}

// GetMarket gets a market by condition_id
// Based on: py-clob-client-main/py_clob_client/client.py:721-725
func (c *ClobClient) GetMarket(conditionID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s%s%s", c.host, types.GET_MARKET, conditionID)
	return c.httpClient.Get(url, nil)
}

// GetMarketTradesEvents gets the market's trades events by condition id
// Based on: py-clob-client-main/py_clob_client/client.py:727-731
func (c *ClobClient) GetMarketTradesEvents(conditionID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s%s%s", c.host, types.GET_MARKET_TRADES_EVENTS, conditionID)
	return c.httpClient.Get(url, nil)
}

// GetAllMarkets gets all markets with pagination
// Helper method to iterate through all markets
func (c *ClobClient) GetAllMarkets() ([]types.Market, error) {
	var allMarkets []types.Market
	cursor := "MA=="

	for cursor != types.EndCursor {
		response, err := c.GetMarkets(cursor)
		if err != nil {
			return nil, err
		}

		// Parse next cursor
		if nextCursor, ok := response["next_cursor"].(string); ok {
			cursor = nextCursor
		} else {
			break
		}

		// Parse markets
		if data, ok := response["data"].([]interface{}); ok {
			for _, item := range data {
				marketJSON, _ := json.Marshal(item)
				var market types.Market
				if err := json.Unmarshal(marketJSON, &market); err == nil {
					allMarkets = append(allMarkets, market)
				}
			}
		}
	}

	return allMarkets, nil
}
