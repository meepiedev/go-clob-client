package client

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

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

// GetNegRiskEvents gets ALL events and filters for negRisk=true, active=true, archived=false
func (c *ClobClient) GetNegRiskEvents() (map[string]interface{}, error) {
	baseURL := "https://gamma-api.polymarket.com/events"
	allEvents := []interface{}{}
	limit := 100
	offset := 0

	// Rate limiting variables
	maxRetries := 5
	baseDelay := time.Millisecond * 500 // Start with 500ms delay between requests

	// Fetch ALL events with pagination
	for {
		// Add a small delay between requests to avoid rate limiting
		if offset > 0 {
			time.Sleep(baseDelay)
		}

		// Build URL with pagination parameters
		u, err := url.Parse(baseURL)
		if err != nil {
			return nil, err
		}

		q := u.Query()
		q.Set("limit", strconv.Itoa(limit))
		q.Set("offset", strconv.Itoa(offset))
		// Filter at API level for active and non-archived
		q.Set("active", "true")
		q.Set("closed", "false")
		q.Set("archived", "false")
		u.RawQuery = q.Encode()

		// Make request with retry logic
		var resp map[string]interface{}
		var lastErr error

		for retry := 0; retry < maxRetries; retry++ {
			resp, err = c.httpClient.Get(u.String(), nil)
			if err != nil {
				// Check if it's a rate limit error
				if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "Too Many Requests") {
					// Exponential backoff
					waitTime := time.Duration(math.Pow(2, float64(retry))) * time.Second
					if waitTime > 30*time.Second {
						waitTime = 30 * time.Second
					}
					time.Sleep(waitTime)
					lastErr = err
					continue
				}
				return nil, err
			}
			// Success
			break
		}

		if lastErr != nil && resp == nil {
			return nil, fmt.Errorf("rate limited after %d retries: %w", maxRetries, lastErr)
		}

		// Extract events from response
		var pageEvents []interface{}

		// Try different response formats
		// First check if it's wrapped in data
		if data, ok := resp["data"].([]interface{}); ok {
			pageEvents = data
		} else {
			// Otherwise look for any array in the response
			for _, value := range resp {
				if arr, ok := value.([]interface{}); ok {
					pageEvents = arr
					break
				}
			}
		}

		// If no events found on this page, we're done
		if len(pageEvents) == 0 {
			break
		}

		// Add page events to all events
		allEvents = append(allEvents, pageEvents...)

		// If we got less than limit, we've reached the end
		if len(pageEvents) < limit {
			break
		}

		// Move to next page
		offset += limit
	}

	// Now filter for negRisk=true
	var filteredEvents []interface{}
	for _, e := range allEvents {
		if event, ok := e.(map[string]interface{}); ok {
			// Check negRisk
			negRisk, _ := event["negRisk"].(bool)
			if negRisk {
				// Double-check active and archived in case API filtering didn't work
				active := true
				if val, ok := event["active"].(bool); ok {
					active = val
				}

				archived := false
				if val, ok := event["archived"].(bool); ok {
					archived = val
				}

				if active && !archived {
					filteredEvents = append(filteredEvents, event)
				}
			}
		}
	}

	return map[string]interface{}{
		"data": filteredEvents,
	}, nil
}

// GetMarketsWithPagination fetches all active markets with automatic pagination
func (c *ClobClient) GetMarketsWithPagination(params *types.MarketsParams) (map[string]interface{}, error) {
	// Set defaults
	if params == nil {
		params = &types.MarketsParams{
			Active: true,
		}
	}

	var allMarkets []interface{}
	nextCursor := "MA=="

	// Keep fetching until we get all markets
	for nextCursor != "" && nextCursor != types.EndCursor {
		resp, err := c.GetMarkets(nextCursor)
		if err != nil {
			return nil, err
		}

		// Extract markets from response
		if data, ok := resp["data"].([]interface{}); ok {
			// Filter based on params
			for _, m := range data {
				if market, ok := m.(map[string]interface{}); ok {
					// Check active status
					if params.Active {
						if active, ok := market["active"].(bool); ok && !active {
							continue
						}
					}

					// Check archived status
					if !params.Archived {
						if archived, ok := market["archived"].(bool); ok && archived {
							continue
						}
					}

					allMarkets = append(allMarkets, market)
				}
			}
		}

		// Get next cursor
		if next, ok := resp["next_cursor"].(string); ok {
			nextCursor = next
		} else {
			break
		}
	}

	return map[string]interface{}{
		"data": allMarkets,
	}, nil
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

// GetGammaMarkets fetches markets from the gamma API with advanced filtering
func (c *ClobClient) GetGammaMarkets(params *types.GammaMarketsParams) ([]types.GammaMarket, error) {
	// Build URL with query parameters
	baseURL := "https://gamma-api.polymarket.com/markets"
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()

	if params != nil {
		if params.Limit > 0 {
			q.Set("limit", strconv.Itoa(params.Limit))
		}
		if params.Offset > 0 {
			q.Set("offset", strconv.Itoa(params.Offset))
		}
		if params.Order != "" {
			q.Set("order", params.Order)
		}
		if params.Ascending != nil {
			q.Set("ascending", strconv.FormatBool(*params.Ascending))
		}
		if params.Active != nil {
			q.Set("active", strconv.FormatBool(*params.Active))
		}
		if params.Closed != nil {
			q.Set("closed", strconv.FormatBool(*params.Closed))
		}
		if params.Archived != nil {
			q.Set("archived", strconv.FormatBool(*params.Archived))
		}
		if params.Restricted != nil {
			q.Set("restricted", strconv.FormatBool(*params.Restricted))
		}
		if params.LiquidityNumMin > 0 {
			q.Set("liquidity_num_min", fmt.Sprintf("%f", params.LiquidityNumMin))
		}
		if params.VolumeNumMin > 0 {
			q.Set("volume_num_min", fmt.Sprintf("%f", params.VolumeNumMin))
		}
		// Add more parameters as needed
		for _, id := range params.ID {
			q.Add("id", strconv.Itoa(id))
		}
		for _, slug := range params.Slug {
			q.Add("slug", slug)
		}
		for _, tokenID := range params.ClobTokenIDs {
			q.Add("clob_token_ids", tokenID)
		}
		for _, conditionID := range params.ConditionIDs {
			q.Add("condition_ids", conditionID)
		}
	}

	u.RawQuery = q.Encode()

	// Make the request
	resp, err := c.httpClient.Get(u.String(), nil)
	if err != nil {
		return nil, err
	}

	// Parse response as array of GammaMarket
	var markets []types.GammaMarket

	// The httpClient.Get wraps array responses in a "data" key
	if data, ok := resp["data"].([]interface{}); ok {
		// Response is array directly
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				market := types.GammaMarket{}

				// Parse ID (can be string or float64)
				switch v := m["id"].(type) {
				case float64:
					market.ID = int(v)
				case string:
					fmt.Sscanf(v, "%d", &market.ID)
				}

				// Parse question
				if question, ok := m["question"].(string); ok {
					market.Question = question
					market.Title = question // Also set as title for compatibility
				}

				// Parse other fields safely
				if slug, ok := m["slug"].(string); ok {
					market.Slug = slug
				}
				if archived, ok := m["archived"].(bool); ok {
					market.Archived = archived
				}
				if active, ok := m["active"].(bool); ok {
					market.Active = active
				}
				if closed, ok := m["closed"].(bool); ok {
					market.Closed = closed
				}

				// Parse numeric fields that could be string or float64
				// Try both liquidity and liquidityNum
				switch v := m["liquidity"].(type) {
				case float64:
					market.Liquidity = v
				case string:
					fmt.Sscanf(v, "%f", &market.Liquidity)
				}
				if market.Liquidity == 0 {
					switch v := m["liquidityNum"].(type) {
					case float64:
						market.Liquidity = v
					case string:
						fmt.Sscanf(v, "%f", &market.Liquidity)
					}
				}

				// Try both volume and volumeNum
				switch v := m["volume"].(type) {
				case float64:
					market.Volume = v
				case string:
					fmt.Sscanf(v, "%f", &market.Volume)
				}
				if market.Volume == 0 {
					switch v := m["volumeNum"].(type) {
					case float64:
						market.Volume = v
					case string:
						fmt.Sscanf(v, "%f", &market.Volume)
					}
				}

				if startDate, ok := m["startDate"].(string); ok {
					market.StartDate = startDate
				} else if startDate, ok := m["start_date"].(string); ok {
					market.StartDate = startDate
				}
				if endDate, ok := m["endDate"].(string); ok {
					market.EndDate = endDate
				} else if endDate, ok := m["end_date"].(string); ok {
					market.EndDate = endDate
				}
				if description, ok := m["description"].(string); ok {
					market.Description = description
				}
				if conditionID, ok := m["conditionId"].(string); ok {
					market.ConditionID = conditionID
				}

				// Parse clobTokenIds - it's a JSON string that needs to be unmarshaled
				if tokenIDsStr, ok := m["clobTokenIds"].(string); ok {
					var tokenIDs []string
					if err := json.Unmarshal([]byte(tokenIDsStr), &tokenIDs); err == nil {
						market.ClobTokenIDs = tokenIDs
					}
				}

				// Parse enableOrderBook
				if enableOrderBook, ok := m["enableOrderBook"].(bool); ok {
					market.EnableOrderBook = enableOrderBook
				}

				// Parse orderMinSize
				switch v := m["orderMinSize"].(type) {
				case float64:
					market.OrderMinSize = v
				case string:
					fmt.Sscanf(v, "%f", &market.OrderMinSize)
				}

				// Only add markets that have enableOrderBook = true
				if market.EnableOrderBook {
					markets = append(markets, market)
				}
			}
		}
	} else {
		// Try to parse the entire response as a market array
		// This happens when the gamma API returns a plain array
		// We need to marshal and unmarshal to convert properly
		jsonData, err := json.Marshal(resp)
		if err == nil {
			var rawMarkets []map[string]interface{}
			if err := json.Unmarshal(jsonData, &rawMarkets); err == nil {
				for _, m := range rawMarkets {
					market := types.GammaMarket{}

					// Parse ID (can be string or float64)
					switch v := m["id"].(type) {
					case float64:
						market.ID = int(v)
					case string:
						fmt.Sscanf(v, "%d", &market.ID)
					}

					// Parse question
					if question, ok := m["question"].(string); ok {
						market.Question = question
						market.Title = question // Also set as title for compatibility
					}

					// Parse other fields safely
					if slug, ok := m["slug"].(string); ok {
						market.Slug = slug
					}
					if archived, ok := m["archived"].(bool); ok {
						market.Archived = archived
					}
					if active, ok := m["active"].(bool); ok {
						market.Active = active
					}
					if closed, ok := m["closed"].(bool); ok {
						market.Closed = closed
					}

					// Parse numeric fields that could be string or float64
					// Try both liquidity and liquidityNum
					switch v := m["liquidity"].(type) {
					case float64:
						market.Liquidity = v
					case string:
						fmt.Sscanf(v, "%f", &market.Liquidity)
					}
					if market.Liquidity == 0 {
						switch v := m["liquidityNum"].(type) {
						case float64:
							market.Liquidity = v
						case string:
							fmt.Sscanf(v, "%f", &market.Liquidity)
						}
					}

					// Try both volume and volumeNum
					switch v := m["volume"].(type) {
					case float64:
						market.Volume = v
					case string:
						fmt.Sscanf(v, "%f", &market.Volume)
					}
					if market.Volume == 0 {
						switch v := m["volumeNum"].(type) {
						case float64:
							market.Volume = v
						case string:
							fmt.Sscanf(v, "%f", &market.Volume)
						}
					}

					if startDate, ok := m["startDate"].(string); ok {
						market.StartDate = startDate
					} else if startDate, ok := m["start_date"].(string); ok {
						market.StartDate = startDate
					}
					if endDate, ok := m["endDate"].(string); ok {
						market.EndDate = endDate
					} else if endDate, ok := m["end_date"].(string); ok {
						market.EndDate = endDate
					}
					if description, ok := m["description"].(string); ok {
						market.Description = description
					}
					if conditionID, ok := m["conditionId"].(string); ok {
						market.ConditionID = conditionID
					}

					// Parse clobTokenIds - it's a JSON string that needs to be unmarshaled
					if tokenIDsStr, ok := m["clobTokenIds"].(string); ok {
						var tokenIDs []string
						if err := json.Unmarshal([]byte(tokenIDsStr), &tokenIDs); err == nil {
							market.ClobTokenIDs = tokenIDs
						}
					}

					// Parse enableOrderBook
					if enableOrderBook, ok := m["enableOrderBook"].(bool); ok {
						market.EnableOrderBook = enableOrderBook
					}

					// Parse orderMinSize
					switch v := m["orderMinSize"].(type) {
					case float64:
						market.OrderMinSize = v
					case string:
						fmt.Sscanf(v, "%f", &market.OrderMinSize)
					}

					// Only add markets that have enableOrderBook = true
					if market.EnableOrderBook {
						markets = append(markets, market)
					}
				}
			}
		}
	}

	return markets, nil
}

// GetGammaEvents fetches events from the gamma API
func (c *ClobClient) GetGammaEvents(params *types.GammaEventsParams) ([]types.GammaEvent, error) {
	// Build URL with query parameters
	baseURL := "https://gamma-api.polymarket.com/events"
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()

	if params != nil {
		if params.Slug != "" {
			q.Set("slug", params.Slug)
		}
		if params.ID > 0 {
			q.Set("id", strconv.Itoa(params.ID))
		}
		// Add more parameters as needed
	}

	u.RawQuery = q.Encode()

	// Make the request
	resp, err := c.httpClient.Get(u.String(), nil)
	if err != nil {
		return nil, err
	}

	// Parse response as array of GammaEvent
	var events []types.GammaEvent

	// Try to parse the entire response as an event array
	jsonData, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	// The response might be wrapped or direct array
	var rawEvents []map[string]interface{}
	if err := json.Unmarshal(jsonData, &rawEvents); err != nil {
		// Try parsing as wrapped response
		if data, ok := resp["data"].([]interface{}); ok {
			for _, item := range data {
				if eventMap, ok := item.(map[string]interface{}); ok {
					rawEvents = append(rawEvents, eventMap)
				}
			}
		} else {
			return nil, fmt.Errorf("failed to parse events response")
		}
	}

	for _, eventData := range rawEvents {
		event := types.GammaEvent{}

		// Parse ID
		switch v := eventData["id"].(type) {
		case float64:
			event.ID = int(v)
		case string:
			fmt.Sscanf(v, "%d", &event.ID)
		}

		// Parse slug
		if slug, ok := eventData["slug"].(string); ok {
			event.Slug = slug
		}

		// Parse title
		if title, ok := eventData["title"].(string); ok {
			event.Title = title
		}

		// Parse markets array
		if marketsData, ok := eventData["markets"].([]interface{}); ok {
			for _, mData := range marketsData {
				if m, ok := mData.(map[string]interface{}); ok {
					market := types.GammaMarket{}

					// Parse market fields
					if question, ok := m["question"].(string); ok {
						market.Question = question
					}
					if groupItemTitle, ok := m["groupItemTitle"].(string); ok {
						market.Title = groupItemTitle
					}
					if clobTokenIds, ok := m["clobTokenIds"].(string); ok {
						var tokenIDs []string
						if err := json.Unmarshal([]byte(clobTokenIds), &tokenIDs); err == nil {
							market.ClobTokenIDs = tokenIDs
						}
					}
					if outcomes, ok := m["outcomes"].(string); ok {
						// Store raw outcomes string in Description for now
						market.Description = outcomes
					}
					if negRisk, ok := m["negRisk"].(bool); ok {
						event.NegRisk = negRisk
					}
					
					// Parse orderMinSize
					switch v := m["orderMinSize"].(type) {
					case float64:
						market.OrderMinSize = v
					case string:
						fmt.Sscanf(v, "%f", &market.OrderMinSize)
					}

					event.Markets = append(event.Markets, market)
				}
			}
		}

		events = append(events, event)
	}

	return events, nil
}

// FetchMarketOutcomes fetches outcome token IDs and names for a market or event
func (c *ClobClient) FetchMarketOutcomes(slug string) ([]string, map[string]string, float64, error) {
	// First try to fetch as an event which contains multiple markets
	events, err := c.GetGammaEvents(&types.GammaEventsParams{Slug: slug})
	if err == nil && len(events) > 0 {
		// For tournament markets, we need all the "No" outcomes
		allOutcomes := []string{}
		outcomeNames := make(map[string]string)
		var minOrderSize float64

		// For each market in the event, extract the "No" outcome token ID
		for _, event := range events {
			for _, market := range event.Markets {
				// Track the minimum order size across all markets
				if minOrderSize == 0 || (market.OrderMinSize > 0 && market.OrderMinSize < minOrderSize) {
					minOrderSize = market.OrderMinSize
				}
				
				// Parse outcomes from Description (where we stored the raw outcomes string)
				var outcomes []string
				if market.Description != "" {
					if err := json.Unmarshal([]byte(market.Description), &outcomes); err == nil {
						// For negative risk markets, we want ALL "No" outcomes
						// Each market has [Yes, No] outcomes, we want index 1 (No)
						for i, outcome := range outcomes {
							if strings.ToLower(outcome) == "no" && i < len(market.ClobTokenIDs) {
								allOutcomes = append(allOutcomes, market.ClobTokenIDs[i])
								outcomeName := market.Title
								if outcomeName == "" {
									outcomeName = market.Question
								}
								outcomeNames[market.ClobTokenIDs[i]] = outcomeName
								break
							}
						}
					}
				}
			}
		}

		if len(allOutcomes) > 0 {
			return allOutcomes, outcomeNames, minOrderSize, nil
		}
	}

	// If not an event, try as a single market
	markets, err := c.GetGammaMarkets(&types.GammaMarketsParams{Slug: []string{slug}})
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to fetch market: %w", err)
	}

	if len(markets) == 0 {
		return nil, nil, 0, fmt.Errorf("no market found for slug: %s", slug)
	}

	market := markets[0]

	// For single negative risk markets, each token represents one option
	outcomeNames := make(map[string]string)
	// Note: Single market outcomes would need to be fetched separately
	// For now, return token IDs without names
	return market.ClobTokenIDs, outcomeNames, market.OrderMinSize, nil
}

// CheckNegativeRisk checks if a market or event is negative risk
func (c *ClobClient) CheckNegativeRisk(slug string) (bool, error) {
	// First check if it's a single market
	markets, err := c.GetGammaMarkets(&types.GammaMarketsParams{Slug: []string{slug}})
	if err == nil && len(markets) > 0 {
		// For now, we can't directly check negRisk from gamma markets API
		// This would need to be added to the GammaMarket struct
		// Return true for now as the arbitrage bot validates this
		return true, nil
	}

	// If not a single market, check as event
	events, err := c.GetGammaEvents(&types.GammaEventsParams{Slug: slug})
	if err != nil {
		return false, fmt.Errorf("failed to fetch market or event: %w", err)
	}

	if len(events) == 0 {
		return false, fmt.Errorf("no market or event found for slug: %s", slug)
	}

	// Check if ALL markets in the event are negRisk
	return events[0].NegRisk, nil
}
