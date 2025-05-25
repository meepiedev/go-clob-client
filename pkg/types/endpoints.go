package types

// API endpoints
// Based on: py-clob-client-main/py_clob_client/endpoints.py
const (
	// Time endpoint
	// Based on: py-clob-client-main/py_clob_client/endpoints.py:1
	TIME = "/time"
	
	// Auth endpoints
	// Based on: py-clob-client-main/py_clob_client/endpoints.py:2-6
	CREATE_API_KEY = "/auth/api-key"
	GET_API_KEYS   = "/auth/api-keys"
	DELETE_API_KEY = "/auth/api-key"
	DERIVE_API_KEY = "/auth/derive-api-key"
	CLOSED_ONLY    = "/auth/ban-status/closed-only"
	
	// Trading data endpoints
	// Based on: py-clob-client-main/py_clob_client/endpoints.py:7-11
	TRADES        = "/data/trades"
	GET_ORDER     = "/data/order/"
	ORDERS        = "/data/orders"
	
	// Order management endpoints
	// Based on: py-clob-client-main/py_clob_client/endpoints.py:12-16
	POST_ORDER           = "/order"
	CANCEL               = "/order"
	CANCEL_ORDERS        = "/orders"
	CANCEL_ALL           = "/cancel-all"
	CANCEL_MARKET_ORDERS = "/cancel-market-orders"
	
	// Book and pricing endpoints
	// Based on: py-clob-client-main/py_clob_client/endpoints.py:8-9,17-24
	GET_ORDER_BOOK        = "/book"
	GET_ORDER_BOOKS       = "/books"
	MID_POINT             = "/midpoint"
	MID_POINTS            = "/midpoints"
	PRICE                 = "/price"
	GET_PRICES            = "/prices"
	GET_SPREAD            = "/spread"
	GET_SPREADS           = "/spreads"
	GET_LAST_TRADE_PRICE  = "/last-trade-price"
	GET_LAST_TRADES_PRICES = "/last-trades-prices"
	
	// Notification endpoints
	// Based on: py-clob-client-main/py_clob_client/endpoints.py:25-26
	GET_NOTIFICATIONS  = "/notifications"
	DROP_NOTIFICATIONS = "/notifications"
	
	// Balance and allowance endpoints
	// Based on: py-clob-client-main/py_clob_client/endpoints.py:27-28
	GET_BALANCE_ALLOWANCE    = "/balance-allowance"
	UPDATE_BALANCE_ALLOWANCE = "/balance-allowance/update"
	
	// Order scoring endpoints
	// Based on: py-clob-client-main/py_clob_client/endpoints.py:29-30
	IS_ORDER_SCORING   = "/order-scoring"
	ARE_ORDERS_SCORING = "/orders-scoring"
	
	// Market info endpoints
	// Based on: py-clob-client-main/py_clob_client/endpoints.py:31-32
	GET_TICK_SIZE = "/tick-size"
	GET_NEG_RISK  = "/neg-risk"
	
	// Market endpoints
	// Based on: py-clob-client-main/py_clob_client/endpoints.py:33-38
	GET_SAMPLING_SIMPLIFIED_MARKETS = "/sampling-simplified-markets"
	GET_SAMPLING_MARKETS            = "/sampling-markets"
	GET_SIMPLIFIED_MARKETS          = "/simplified-markets"
	GET_MARKETS                     = "/markets"
	GET_MARKET                      = "/markets/"
	GET_MARKET_TRADES_EVENTS        = "/live-activity/events/"
)