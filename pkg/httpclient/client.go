package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	
	"github.com/pooofdevelopment/go-clob-client/pkg/types"
)

// Client wraps the standard HTTP client with common functionality
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new HTTP client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Get performs a GET request
// Based on: py-clob-client-main/py_clob_client/http_helpers/helpers.py:50-60
func (c *Client) Get(url string, headers map[string]string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	// Add headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	return c.parseResponse(resp)
}

// Post performs a POST request
// Based on: py-clob-client-main/py_clob_client/http_helpers/helpers.py:63-73
func (c *Client) Post(url string, headers map[string]string, data interface{}) (map[string]interface{}, error) {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(jsonData)
	}
	
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	
	// Set content type
	req.Header.Set("Content-Type", "application/json")
	
	// Add headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	return c.parseResponse(resp)
}

// Delete performs a DELETE request
// Based on: py-clob-client-main/py_clob_client/http_helpers/helpers.py:76-86
func (c *Client) Delete(url string, headers map[string]string, data interface{}) (map[string]interface{}, error) {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(jsonData)
	}
	
	req, err := http.NewRequest("DELETE", url, body)
	if err != nil {
		return nil, err
	}
	
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	// Add headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	return c.parseResponse(resp)
}

// parseResponse parses the HTTP response
// Based on: py-clob-client-main/py_clob_client/http_helpers/helpers.py:35-47
func (c *Client) parseResponse(resp *http.Response) (map[string]interface{}, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	// Check for non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errorData map[string]interface{}
		if err := json.Unmarshal(body, &errorData); err != nil {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}
		
		// Try to extract error message
		if msg, ok := errorData["error"].(string); ok {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
		} else if msg, ok := errorData["message"].(string); ok {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
		}
		
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	
	// First try to parse as JSON object
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		// Try to parse as array
		var arrayResult []interface{}
		if err := json.Unmarshal(body, &arrayResult); err != nil {
			// If it's not JSON, check if it's a simple string response
			bodyStr := string(body)
			if bodyStr != "" {
				// For simple string responses like "OK", wrap in a map
				return map[string]interface{}{"message": strings.Trim(bodyStr, "\"")}, nil
			}
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		// Wrap array in a map
		result = map[string]interface{}{"data": arrayResult}
	}
	
	return result, nil
}

// AddQueryTradeParams adds trade parameters to URL
// Based on: py-clob-client-main/py_clob_client/http_helpers/helpers.py:8-20
func AddQueryTradeParams(baseURL string, params *types.TradeParams, nextCursor string) string {
	u, _ := url.Parse(baseURL)
	q := u.Query()
	
	if nextCursor != "" {
		q.Set("next_cursor", nextCursor)
	}
	
	if params != nil {
		if params.ID != "" {
			q.Set("id", params.ID)
		}
		if params.MakerAddress != "" {
			q.Set("maker_address", params.MakerAddress)
		}
		if params.Market != "" {
			q.Set("market", params.Market)
		}
		if params.AssetID != "" {
			q.Set("asset_id", params.AssetID)
		}
		if params.Before > 0 {
			q.Set("before", strconv.FormatInt(params.Before, 10))
		}
		if params.After > 0 {
			q.Set("after", strconv.FormatInt(params.After, 10))
		}
	}
	
	u.RawQuery = q.Encode()
	return u.String()
}

// AddQueryOpenOrdersParams adds open order parameters to URL
// Based on: py-clob-client-main/py_clob_client/http_helpers/helpers.py:23-32
func AddQueryOpenOrdersParams(baseURL string, params *types.OpenOrderParams, nextCursor string) string {
	u, _ := url.Parse(baseURL)
	q := u.Query()
	
	if nextCursor != "" {
		q.Set("next_cursor", nextCursor)
	}
	
	if params != nil {
		if params.ID != "" {
			q.Set("id", params.ID)
		}
		if params.Market != "" {
			q.Set("market", params.Market)
		}
		if params.AssetID != "" {
			q.Set("asset_id", params.AssetID)
		}
	}
	
	u.RawQuery = q.Encode()
	return u.String()
}

// DropNotificationsQueryParams builds URL for drop notifications
// Based on: py-clob-client-main/py_clob_client/http_helpers/helpers.py:89-93
func DropNotificationsQueryParams(baseURL string, params *types.DropNotificationParams) string {
	if params == nil || len(params.IDs) == 0 {
		return baseURL
	}
	
	u, _ := url.Parse(baseURL)
	q := u.Query()
	q.Set("ids", strings.Join(params.IDs, ","))
	u.RawQuery = q.Encode()
	return u.String()
}

// AddBalanceAllowanceParamsToURL adds balance/allowance parameters to URL
// Based on: py-clob-client-main/py_clob_client/http_helpers/helpers.py:96-104
func AddBalanceAllowanceParamsToURL(baseURL string, params *types.BalanceAllowanceParams) string {
	if params == nil {
		return baseURL
	}
	
	u, _ := url.Parse(baseURL)
	q := u.Query()
	
	if params.AssetType != "" {
		q.Set("asset_type", string(params.AssetType))
	}
	if params.TokenID != "" {
		q.Set("token_id", params.TokenID)
	}
	if params.SignatureType >= 0 {
		q.Set("signature_type", strconv.Itoa(params.SignatureType))
	}
	
	u.RawQuery = q.Encode()
	return u.String()
}

// AddOrderScoringParamsToURL adds order scoring parameters to URL
// Based on: py-clob-client-main/py_clob_client/http_helpers/helpers.py:107-111
func AddOrderScoringParamsToURL(baseURL string, params *types.OrderScoringParams) string {
	if params == nil {
		return baseURL
	}
	
	u, _ := url.Parse(baseURL)
	q := u.Query()
	
	if params.OrderID != "" {
		q.Set("orderId", params.OrderID)
	}
	
	u.RawQuery = q.Encode()
	return u.String()
}