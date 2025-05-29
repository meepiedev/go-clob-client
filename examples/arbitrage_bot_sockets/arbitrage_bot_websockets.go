package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/pooofdevelopment/go-clob-client/pkg/client"
	"github.com/pooofdevelopment/go-clob-client/pkg/types"
	"github.com/pooofdevelopment/go-clob-client/pkg/websocket"
)

// BotConfig represents the arbitrage bot configuration
type BotConfig struct {
	APIKey        string                    `json:"api_key,omitempty"`
	APISecret     string                    `json:"api_secret,omitempty"`
	APIPassphrase string                    `json:"api_passphrase,omitempty"`
	PrivateKey    string                    `json:"private_key"`
	Address       string                    `json:"address"`
	SignatureType string                    `json:"signature_type"` // "eoa" or "poly_proxy"
	MaxSpendUSDC  float64                   `json:"max_spend_usdc"`
	Markets       []MarketGroup             `json:"markets"`
	Strategies    map[string]StrategyConfig `json:"strategies"`
}

// MarketGroup represents a group of mutually exclusive outcomes
type MarketGroup struct {
	Name         string            `json:"name"`
	Slug         string            `json:"slug,omitempty"`     // Optional: fetch outcomes from API
	Outcomes     []string          `json:"outcomes,omitempty"` // Token IDs (auto-populated if slug provided)
	OutcomeNames map[string]string `json:"-"`                  // Token ID -> outcome name mapping
}

// StrategyConfig represents configuration for each strategy
type StrategyConfig struct {
	Enabled      bool    `json:"enabled"`
	MinEdge      float64 `json:"min_edge"`
	WorkerAmount int     `json:"worker_amount"`
	PollInterval int     `json:"poll_interval_ms"`
	ExtraEdge    float64 `json:"extra_edge,omitempty"` // For maker-taker only
}

// ArbBot represents the arbitrage bot instance
type ArbBot struct {
	client        *client.ClobClient
	config        BotConfig
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	activeBids    map[string]map[string]*ActiveBid // [marketGroup][outcome] -> bid
	bidsMutex     sync.RWMutex
	stats         *Stats
	metrics       *PerformanceMetrics
	screen        tcell.Screen
	uiMutex       sync.Mutex
	uiData        *UIData
	marketCache   *MarketCache
	arbTrigger    chan string             // Channel to trigger immediate arbitrage checks
	tokenToMarket map[string]*MarketGroup // Pre-computed mapping for O(1) lookup
	arbWorkers    []*ArbWorker            // Pre-allocated worker pool
}

// ArbWorker represents a pre-allocated worker for arbitrage checking
type ArbWorker struct {
	bot      *ArbBot
	cache    *MarketCache
	trigger  chan string
	marketID int
}

// ActiveBid tracks an active limit order
type ActiveBid struct {
	OrderID string
	Price   float64
	Size    float64
}

// MarketData holds orderbook data for quick access
type MarketData struct {
	BestAsk     float64
	BestAskSize float64
	BestBid     float64
	BestBidSize float64
	LastUpdate  time.Time
}

// Stats tracks arbitrage bot performance
type Stats struct {
	opportunitiesFound int64
	ordersPlaced       int64
	ordersSucceeded    int64
	ordersFailed       int64
	startTime          time.Time
}

// PerformanceMetrics tracks latency and performance
type PerformanceMetrics struct {
	captureToOrderLatencies []time.Duration
	orderExecutionTimes     []time.Duration
	mutex                   sync.RWMutex
}

// MarketCache provides thread-safe market data caching
type MarketCache struct {
	data sync.Map // Lock-free concurrent map
}

// UIData holds data for the terminal UI display
type UIData struct {
	marketGroups []MarketGroupUI
	lastUpdate   time.Time
	startTime    time.Time
}

// MarketGroupUI represents UI data for a market group
type MarketGroupUI struct {
	name          string
	outcomes      []OutcomeUI
	totalCost     float64
	arbitrage     float64
	isOpportunity bool
}

// OutcomeUI represents UI data for a single outcome
type OutcomeUI struct {
	tokenID     string
	name        string
	bestAsk     float64
	bestAskSize float64
	bestBid     float64
	bestBidSize float64
	activeBid   *ActiveBid
}

func NewMarketCache() *MarketCache {
	return &MarketCache{}
}

func (mc *MarketCache) Update(tokenID string, data *MarketData) {
	mc.data.Store(tokenID, data)
}

func (mc *MarketCache) Get(tokenID string) (*MarketData, bool) {
	value, exists := mc.data.Load(tokenID)
	if !exists {
		return nil, false
	}
	return value.(*MarketData), true
}

func (pm *PerformanceMetrics) RecordCaptureToOrder(latency time.Duration) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.captureToOrderLatencies = append(pm.captureToOrderLatencies, latency)
}

func (pm *PerformanceMetrics) RecordOrderExecution(duration time.Duration) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.orderExecutionTimes = append(pm.orderExecutionTimes, duration)
}

func (pm *PerformanceMetrics) GetStats() (avgCaptureToOrder, avgOrderExecution time.Duration, p99CaptureToOrder, p99OrderExecution time.Duration) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	if len(pm.captureToOrderLatencies) > 0 {
		total := time.Duration(0)
		for _, d := range pm.captureToOrderLatencies {
			total += d
		}
		avgCaptureToOrder = total / time.Duration(len(pm.captureToOrderLatencies))

		// Calculate P99
		if len(pm.captureToOrderLatencies) > 100 {
			p99Index := int(float64(len(pm.captureToOrderLatencies)) * 0.99)
			p99CaptureToOrder = pm.captureToOrderLatencies[p99Index]
		}
	}

	if len(pm.orderExecutionTimes) > 0 {
		total := time.Duration(0)
		for _, d := range pm.orderExecutionTimes {
			total += d
		}
		avgOrderExecution = total / time.Duration(len(pm.orderExecutionTimes))

		// Calculate P99
		if len(pm.orderExecutionTimes) > 100 {
			p99Index := int(float64(len(pm.orderExecutionTimes)) * 0.99)
			p99OrderExecution = pm.orderExecutionTimes[p99Index]
		}
	}

	return
}

func main() {
	// Set up logging to file
	logFile, err := os.OpenFile("arbitrage_bot.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// Create a multi-writer to write to both file and console
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	log.Println("=== Arbitrage Bot Starting ===")

	// Load configuration
	configData, err := os.ReadFile("arbitrage_config.json")
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	var cfg BotConfig
	if err := json.Unmarshal(configData, &cfg); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	// Check if markets are negative risk and fetch outcomes
	for i, market := range cfg.Markets {
		if market.Slug != "" {
			// Check if market is negative risk
			log.Printf("Checking if '%s' is a negative risk market...", market.Name)
			// Create a temporary client for fetching market data
			tempClient, _ := client.NewClobClient("https://clob.polymarket.com", 137, "", nil, nil, nil)
			isNegRisk, err := tempClient.CheckNegativeRisk(market.Slug)
			if err != nil {
				log.Printf("Warning: Failed to check negRisk status: %v", err)
			} else if !isNegRisk {
				log.Fatalf("ERROR: Market '%s' is NOT a negative risk market! This bot only works with negative risk markets.", market.Name)
			} else {
				log.Printf("✓ Confirmed: '%s' is a negative risk market", market.Name)
			}

			// Fetch outcomes if not already provided
			if len(market.Outcomes) == 0 {
				log.Printf("Fetching outcomes for market: %s", market.Name)
				outcomes, outcomeNames, err := tempClient.FetchMarketOutcomes(market.Slug)
				if err != nil {
					log.Fatalf("Failed to fetch outcomes for %s: %v", market.Name, err)
				}
				cfg.Markets[i].Outcomes = outcomes
				cfg.Markets[i].OutcomeNames = outcomeNames
				log.Printf("Found %d outcomes for %s", len(outcomes), market.Name)
				// Log token IDs with their names
				for j, tokenID := range outcomes {
					if j < 10 {
						name := outcomeNames[tokenID]
						if name == "" {
							name = "Unknown"
						}
						log.Printf("  Token %d: %s -> %s", j+1, tokenID, name)
					}
				}
			}
		}
	}

	// Create client configuration
	host := "https://clob.polymarket.com"
	chainID := 137 // Polygon mainnet

	var clobClient *client.ClobClient

	// Determine signature type
	var sigType *model.SignatureType
	if strings.ToLower(cfg.SignatureType) == "poly_proxy" {
		st := model.SignatureType(types.POLY_PROXY)
		sigType = &st
		log.Println("Using POLY_PROXY signature type")
	} else {
		st := model.SignatureType(types.EOA)
		sigType = &st
		log.Println("Using EOA signature type")
	}

	// Check if we need to create API credentials
	if cfg.APIKey == "" && cfg.PrivateKey != "" {
		log.Println("API key not provided, creating L2 client with derived keys...")

		// First create client with private key only
		clobClient, err = client.NewClobClient(host, chainID, cfg.PrivateKey, nil, sigType, nil)
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}

		// Derive API credentials
		creds, err := clobClient.CreateOrDeriveApiCreds(nil)
		if err != nil {
			log.Fatalf("Failed to derive API credentials: %v", err)
		}

		// Set the credentials on the client
		clobClient.SetApiCreds(creds)

		log.Printf("Generated API credentials successfully")

		// Save the generated credentials back to config
		cfg.APIKey = creds.ApiKey
		cfg.APISecret = creds.ApiSecret
		cfg.APIPassphrase = creds.ApiPassphrase
		saveUpdatedConfig(cfg)

	} else if cfg.APIKey != "" {
		// Create L2 client directly with provided credentials
		log.Println("Creating L2 client with provided API credentials...")

		creds := &types.ApiCreds{
			ApiKey:        cfg.APIKey,
			ApiSecret:     cfg.APISecret,
			ApiPassphrase: cfg.APIPassphrase,
		}

		clobClient, err = client.NewClobClient(host, chainID, cfg.PrivateKey, creds, sigType, nil)
		if err != nil {
			log.Fatalf("Failed to create L2 client: %v", err)
		}
	} else {
		log.Fatalf("Either API credentials or private key must be provided")
	}

	// Verify client is working
	log.Printf("Client initialized for address: %s", clobClient.GetAddress())

	// Create bot instance
	ctx, cancel := context.WithCancel(context.Background())
	bot := &ArbBot{
		client:     clobClient,
		config:     cfg,
		ctx:        ctx,
		cancel:     cancel,
		activeBids: make(map[string]map[string]*ActiveBid),
		stats: &Stats{
			startTime: time.Now(),
		},
		metrics: &PerformanceMetrics{
			captureToOrderLatencies: make([]time.Duration, 0, 10000),
			orderExecutionTimes:     make([]time.Duration, 0, 10000),
		},
		uiData: &UIData{
			startTime: time.Now(),
		},
		arbTrigger:    make(chan string, 1000), // Buffered channel for immediate checks
		tokenToMarket: make(map[string]*MarketGroup),
	}

	// Pre-compute token to market mapping for O(1) lookup
	for i := range cfg.Markets {
		market := &cfg.Markets[i]
		// Initialize OutcomeNames if nil
		if market.OutcomeNames == nil {
			market.OutcomeNames = make(map[string]string)
		}
		for _, tokenID := range market.Outcomes {
			bot.tokenToMarket[tokenID] = market
		}
	}

	// Initialize terminal UI
	if err := bot.initUI(); err != nil {
		log.Fatalf("Failed to initialize UI: %v", err)
	}

	// Start UI update loop
	go bot.runUILoop()

	// Start market data updater
	marketCache := NewMarketCache()
	bot.marketCache = marketCache

	// UI will update via the runUILoop goroutine

	go bot.startMarketDataUpdater(marketCache)

	// Start strategies
	if takerCfg, exists := cfg.Strategies["taker"]; exists && takerCfg.Enabled {
		log.Printf("Starting TAKER strategy with %d workers", takerCfg.WorkerAmount)
		bot.startTakerStrategy(marketCache, takerCfg)
	}

	if makerTakerCfg, exists := cfg.Strategies["maker-taker"]; exists && makerTakerCfg.Enabled {
		log.Printf("Starting MAKER-TAKER strategy with %d workers", makerTakerCfg.WorkerAmount)
		bot.startMakerTakerStrategy(marketCache, makerTakerCfg)
	}

	// Run the main event loop
	bot.runEventLoop()

	// Cleanup
	bot.cleanup()
}

func (bot *ArbBot) initUI() error {
	var err error
	bot.screen, err = tcell.NewScreen()
	if err != nil {
		return err
	}

	if err := bot.screen.Init(); err != nil {
		return err
	}

	bot.screen.Clear()
	return nil
}

func (bot *ArbBot) cleanup() {
	if bot.screen != nil {
		bot.screen.Fini()
	}
	bot.cancel()
	bot.wg.Wait()
}

func (bot *ArbBot) runEventLoop() {
	for {
		ev := bot.screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC || ev.Rune() == 'q' {
				return
			}
		case *tcell.EventResize:
			bot.screen.Sync()
			bot.drawUI()
		}
	}
}

func (bot *ArbBot) runUILoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-bot.ctx.Done():
			return
		case <-ticker.C:
			bot.updateUIData()
			bot.drawUI()
		}
	}
}

func (bot *ArbBot) updateUIData() {
	bot.uiMutex.Lock()
	defer bot.uiMutex.Unlock()

	// Update market groups with latest data
	bot.uiData.marketGroups = nil
	bot.uiData.lastUpdate = time.Now()

	for _, market := range bot.config.Markets {
		group := MarketGroupUI{
			name:     market.Name,
			outcomes: make([]OutcomeUI, 0, len(market.Outcomes)),
		}

		totalCost := 0.0

		for _, tokenID := range market.Outcomes {
			outcome := OutcomeUI{
				tokenID: tokenID,
			}

			// Get outcome name
			if name, exists := market.OutcomeNames[tokenID]; exists {
				outcome.name = name
			}

			// Get market data from cache
			if data, exists := bot.marketCache.Get(tokenID); exists {
				outcome.bestAsk = data.BestAsk
				outcome.bestAskSize = data.BestAskSize
				outcome.bestBid = data.BestBid
				outcome.bestBidSize = data.BestBidSize

				if data.BestAsk > 0 {
					totalCost += data.BestAsk
				}
			}

			// Check for active bid
			bot.bidsMutex.RLock()
			if marketBids, exists := bot.activeBids[market.Name]; exists {
				outcome.activeBid = marketBids[tokenID]
			}
			bot.bidsMutex.RUnlock()

			group.outcomes = append(group.outcomes, outcome)
		}

		// Always update with current data, even if partial
		group.totalCost = totalCost

		// Calculate arbitrage based on available liquid shares
		liquidCount := 0
		for _, outcome := range group.outcomes {
			if outcome.bestAsk > 0 {
				liquidCount++
			}
		}

		if liquidCount >= 2 {
			k := float64(liquidCount)
			group.arbitrage = (k - 1) - totalCost

			// Check if it's an opportunity based on strategy settings
			if takerCfg, exists := bot.config.Strategies["taker"]; exists && takerCfg.Enabled {
				group.isOpportunity = group.arbitrage > takerCfg.MinEdge
			} else if makerCfg, exists := bot.config.Strategies["maker-taker"]; exists && makerCfg.Enabled {
				group.isOpportunity = group.arbitrage > makerCfg.MinEdge
			}
		} else {
			group.arbitrage = 0
			group.isOpportunity = false
		}

		bot.uiData.marketGroups = append(bot.uiData.marketGroups, group)
	}
}

func (bot *ArbBot) drawUI() {
	bot.uiMutex.Lock()
	defer bot.uiMutex.Unlock()

	bot.screen.Clear()
	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	boldStyle := defStyle.Bold(true)
	greenStyle := defStyle.Foreground(tcell.ColorGreen)
	yellowStyle := defStyle.Foreground(tcell.ColorYellow)
	cyanStyle := defStyle.Foreground(tcell.ColorDarkCyan)

	width, height := bot.screen.Size()
	y := 0

	// Compact header
	header := "Negative Risk Arbitrage Bot"
	bot.drawText(0, y, width, y, boldStyle.Foreground(tcell.ColorBlue), header)
	y++

	// Compact stats on one line
	runtime := time.Since(bot.stats.startTime)
	statsLine := fmt.Sprintf("Runtime: %v | Opps: %d | Orders: %d/%d",
		runtime.Round(time.Second),
		atomic.LoadInt64(&bot.stats.opportunitiesFound),
		atomic.LoadInt64(&bot.stats.ordersSucceeded),
		atomic.LoadInt64(&bot.stats.ordersPlaced))
	bot.drawText(0, y, width, y, defStyle, statsLine)
	y += 2

	// Market summary for each group
	for _, group := range bot.uiData.marketGroups {
		validCount := 0
		for _, outcome := range group.outcomes {
			if outcome.bestAsk > 0 {
				validCount++
			}
		}

		// Market name and status
		marketLine := fmt.Sprintf("%s (%d/%d valid)", group.name, validCount, len(group.outcomes))
		bot.drawText(0, y, width, y, cyanStyle, marketLine)
		y++

		// Show aggregate data
		if validCount >= 2 { // Need at least 2 for arbitrage
			k := float64(validCount)
			guaranteedPayout := k - 1
			edge := guaranteedPayout - group.totalCost

			totalCostStr := fmt.Sprintf("Total: $%.4f for %d shares", group.totalCost, validCount)
			payoutStr := fmt.Sprintf("Guaranteed: $%.4f", guaranteedPayout)
			edgeStr := fmt.Sprintf("Edge: $%.4f (%.2f%%)", edge, (edge/guaranteedPayout)*100)

			lineStyle := defStyle
			statusStr := ""

			// Check if this is an opportunity based on actual liquid shares
			if edge > 0.01 { // Using min edge threshold
				lineStyle = greenStyle.Bold(true)
				statusStr = " 🚨 OPPORTUNITY!"
			}

			summaryLine := fmt.Sprintf("%s | %s | %s%s", totalCostStr, payoutStr, edgeStr, statusStr)
			bot.drawText(0, y, width, y, lineStyle, summaryLine)

			if len(group.outcomes)-validCount > 0 {
				skipLine := fmt.Sprintf("  (Skipping %d illiquid outcomes)", len(group.outcomes)-validCount)
				bot.drawText(0, y+1, width, y+1, defStyle.Foreground(tcell.ColorDimGray), skipLine)
				y++
			}
		} else {
			waitingLine := fmt.Sprintf("Only %d liquid outcome(s) - need at least 2 for arbitrage", validCount)
			bot.drawText(0, y, width, y, yellowStyle, waitingLine)
		}
		y += 2

		// Show sample prices (max 3 lines)
		if height-y > 5 {
			showCount := 0
			for i, outcome := range group.outcomes {
				if showCount >= 3 || y >= height-3 {
					if showCount < len(group.outcomes) {
						bot.drawText(0, y, width, y, defStyle, fmt.Sprintf("  ... and %d more", len(group.outcomes)-showCount))
						y++
					}
					break
				}

				if outcome.bestAsk > 0 {
					var line string
					if outcome.name != "" {
						// Truncate long names
						name := outcome.name
						if len(name) > 30 {
							name = name[:27] + "..."
						}
						line = fmt.Sprintf("  [%d] %s: $%.4f (%.1f shares)", i+1, name, outcome.bestAsk, outcome.bestAskSize)
					} else {
						line = fmt.Sprintf("  [%d] $%.4f (%.1f shares)", i+1, outcome.bestAsk, outcome.bestAskSize)
					}
					bot.drawText(0, y, width, y, defStyle, line)
					y++
					showCount++
				}
			}
		}
		y++
	}

	// Footer
	footer := "Press 'q' or ESC to quit"
	bot.drawText(0, height-1, width, height-1, defStyle.Foreground(tcell.ColorDimGray), footer)

	bot.screen.Show()
}

func (bot *ArbBot) drawText(x1, y1, x2, y2 int, style tcell.Style, text string) {
	row := y1
	col := x1
	for _, r := range text {
		bot.screen.SetContent(col, row, r, nil, style)
		col++
		if col >= x2 {
			row++
			col = x1
		}
		if row > y2 {
			break
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// WebSocketHandler implements the websocket.MessageHandler interface
type WebSocketHandler struct {
	cache *MarketCache
	bot   *ArbBot
}

func (h *WebSocketHandler) OnOrderBookUpdate(update *websocket.OrderBookUpdate) {
	// FAST PATH: Extract price data first
	var askPrice, askSize float64
	if update.Asks != nil && len(update.Asks) > 0 {
		askPrice, _ = strconv.ParseFloat(update.Asks[0].Price, 64)
		askSize, _ = strconv.ParseFloat(update.Asks[0].Size, 64)
	} else if update.Sells != nil && len(update.Sells) > 0 {
		askPrice, _ = strconv.ParseFloat(update.Sells[0].Price, 64)
		askSize, _ = strconv.ParseFloat(update.Sells[0].Size, 64)
	}

	// Update cache immediately
	data := &MarketData{
		BestAsk:     askPrice,
		BestAskSize: askSize,
		LastUpdate:  time.Now(),
	}

	// Also get bid data
	if update.Bids != nil && len(update.Bids) > 0 {
		data.BestBid, _ = strconv.ParseFloat(update.Bids[0].Price, 64)
		data.BestBidSize, _ = strconv.ParseFloat(update.Bids[0].Size, 64)
	} else if update.Buys != nil && len(update.Buys) > 0 {
		data.BestBid, _ = strconv.ParseFloat(update.Buys[0].Price, 64)
		data.BestBidSize, _ = strconv.ParseFloat(update.Buys[0].Size, 64)
	}

	// Update cache
	h.cache.Update(update.AssetID, data)

	// FAST PATH: Inline arbitrage check if ask price changed
	if askPrice > 0 && h.bot.config.MaxSpendUSDC > 0 {
		// Check which market this token belongs to
		market := h.bot.tokenToMarket[update.AssetID]
		if market != nil {
			// Fast calculation: sum all asks for this market
			totalCost := 0.0
			liquidCount := 0

			for _, tokenID := range market.Outcomes {
				if md, exists := h.cache.Get(tokenID); exists && md.BestAsk > 0 {
					totalCost += md.BestAsk
					liquidCount++
				}
			}

			// Check arbitrage condition inline
			if liquidCount >= 2 {
				k := float64(liquidCount)
				minEdge := 0.001 // Use taker min edge

				if totalCost < (k-1)-minEdge && totalCost <= h.bot.config.MaxSpendUSDC {
					// ARBITRAGE DETECTED - trigger immediately
					log.Printf("[WS FAST] Arbitrage detected in %dus! Market: %s, Cost: %.4f < %.4f",
						time.Since(data.LastUpdate).Microseconds(), market.Name, totalCost, k-1)

					// Send to dedicated arbitrage execution channel
					select {
					case h.bot.arbTrigger <- update.AssetID:
					default:
					}
				}
			}
		}
	}

	// Log every WebSocket message received
	tokenShort := update.AssetID
	if len(tokenShort) > 8 {
		tokenShort = tokenShort[:8] + "..."
	}

	// Get outcome name if available
	outcomeName := ""
	if market := h.bot.tokenToMarket[update.AssetID]; market != nil {
		if name, exists := market.OutcomeNames[update.AssetID]; exists {
			outcomeName = name
		}
	}

	if outcomeName != "" {
		log.Printf("[WS UPDATE] %s (%s): Bid %.4f (%.2f) | Ask %.4f (%.2f)",
			outcomeName, tokenShort, data.BestBid, data.BestBidSize, data.BestAsk, data.BestAskSize)
	} else {
		log.Printf("[WS UPDATE] Token %s: Bid %.4f (%.2f) | Ask %.4f (%.2f)",
			tokenShort, data.BestBid, data.BestBidSize, data.BestAsk, data.BestAskSize)
	}
}

func (h *WebSocketHandler) OnPriceChange(update *websocket.PriceChangeUpdate) {
	// Not needed for arbitrage bot
}

func (h *WebSocketHandler) OnTickSizeChange(update *websocket.TickSizeChangeUpdate) {
	// Not needed for arbitrage bot
}

func (h *WebSocketHandler) OnLastTradePrice(update *websocket.LastTradePriceUpdate) {
	// Last trade price updates can be ignored for arbitrage bot
	// We only care about orderbook best bid/ask prices
}

func (h *WebSocketHandler) OnUserUpdate(update *websocket.UserUpdate) {
	// Not needed for arbitrage bot
}

func (h *WebSocketHandler) OnError(err error) {
	log.Printf("WebSocket error: %v", err)
}

func (h *WebSocketHandler) OnConnect() {
	log.Println("WebSocket connected successfully")
}

func (h *WebSocketHandler) OnDisconnect() {
	log.Println("WebSocket disconnected")
}

func (bot *ArbBot) startMarketDataUpdater(cache *MarketCache) {
	// Collect all unique NO share token IDs
	var tokenIDs []string
	tokenMap := make(map[string]bool)
	for _, market := range bot.config.Markets {
		for _, outcome := range market.Outcomes {
			if !tokenMap[outcome] {
				tokenMap[outcome] = true
				tokenIDs = append(tokenIDs, outcome)
			}
		}
	}

	log.Printf("Starting WebSocket connection for %d NO share tokens", len(tokenIDs))
	log.Println("Monitoring for negative risk arbitrage opportunities...")

	// Create WebSocket handler
	handler := &WebSocketHandler{
		cache: cache,
		bot:   bot,
	}

	// Use the SDK's WebSocket subscription method
	wsClient, err := bot.client.SubscribeToMarketData(tokenIDs, handler)
	if err != nil {
		log.Printf("Failed to subscribe via WebSocket: %v, falling back to polling", err)
		bot.startPollingFallback(cache, tokenIDs)
		return
	}

	// Keep WebSocket alive in a goroutine
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-bot.ctx.Done():
				wsClient.Close()
				return
			case <-ticker.C:
				if !wsClient.IsConnected() {
					log.Println("WebSocket disconnected, attempting to reconnect...")
					// Try to resubscribe
					newClient, err := bot.client.SubscribeToMarketData(tokenIDs, handler)
					if err != nil {
						log.Printf("Reconnection failed: %v", err)
					} else {
						wsClient.Close()
						wsClient = newClient
					}
				}
			}
		}
	}()

	// Also start a verification poller for empty books
	go bot.verifyEmptyBooks(cache, tokenIDs)
}

func (bot *ArbBot) verifyEmptyBooks(cache *MarketCache, tokenIDs []string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-bot.ctx.Done():
			return
		case <-ticker.C:
			// Check for any tokens with no ask data
			emptyTokens := []string{}
			for _, tokenID := range tokenIDs {
				data, exists := cache.Get(tokenID)
				if !exists || data.BestAsk == 0 {
					emptyTokens = append(emptyTokens, tokenID)
				}
			}

			// Double-check empty books via API
			if len(emptyTokens) > 0 {
				for _, tokenID := range emptyTokens {
					book, err := bot.client.GetOrderBook(tokenID)
					if err == nil && len(book.Asks) > 0 {
						askPrice, _ := strconv.ParseFloat(book.Asks[0].Price, 64)
						askSize, _ := strconv.ParseFloat(book.Asks[0].Size, 64)
						if askPrice > 0 {
							data := &MarketData{
								BestAsk:     askPrice,
								BestAskSize: askSize,
								LastUpdate:  time.Now(),
							}
							cache.Update(tokenID, data)
							log.Printf("API verification found ask for previously empty token: %s at $%.4f", tokenID[:8], askPrice)
						}
					}
				}
			}
		}
	}
}

func (bot *ArbBot) startPollingFallback(cache *MarketCache, tokenIDs []string) {
	log.Println("Using polling fallback for market data")

	// Create semaphore to limit concurrent requests
	sem := make(chan struct{}, 10) // Max 10 concurrent requests

	for {
		select {
		case <-bot.ctx.Done():
			return
		default:
			var wg sync.WaitGroup
			for _, tokenID := range tokenIDs {
				wg.Add(1)
				sem <- struct{}{} // Acquire semaphore
				go func(tid string) {
					defer wg.Done()
					defer func() { <-sem }() // Release semaphore

					// Call the client method to get orderbook
					book, err := bot.client.GetOrderBook(tid)
					if err != nil {
						return
					}

					data := &MarketData{
						LastUpdate: time.Now(),
					}

					if len(book.Asks) > 0 {
						askPrice, _ := strconv.ParseFloat(book.Asks[0].Price, 64)
						askSize, _ := strconv.ParseFloat(book.Asks[0].Size, 64)
						data.BestAsk = askPrice
						data.BestAskSize = askSize
					}
					if len(book.Bids) > 0 {
						bidPrice, _ := strconv.ParseFloat(book.Bids[0].Price, 64)
						bidSize, _ := strconv.ParseFloat(book.Bids[0].Size, 64)
						data.BestBid = bidPrice
						data.BestBidSize = bidSize
					}

					cache.Update(tid, data)
				}(tokenID)
			}
			wg.Wait()
			time.Sleep(500 * time.Millisecond) // Less aggressive for fallback
		}
	}
}

func (bot *ArbBot) startTakerStrategy(cache *MarketCache, cfg StrategyConfig) {
	for _, market := range bot.config.Markets {
		for i := 0; i < cfg.WorkerAmount; i++ {
			bot.wg.Add(1)
			go bot.runTakerArb(market, cache, cfg)
		}
	}
}

func (bot *ArbBot) runTakerArb(market MarketGroup, cache *MarketCache, cfg StrategyConfig) {
	defer bot.wg.Done()

	// In a market with n mutually exclusive outcomes:
	// - Exactly one outcome will resolve to YES
	// - If you buy k NO shares, you're guaranteed k-1 payouts
	// - So if sum(k NO prices) < k-1, there's guaranteed profit
	ticker := time.NewTicker(time.Duration(cfg.PollInterval) * time.Millisecond)
	defer ticker.Stop()

	// Function to check arbitrage opportunity
	checkArbitrage := func() {
		if bot.config.MaxSpendUSDC <= 0 {
			log.Printf("[TAKER] MaxSpendUSDC not set or zero, skipping arbitrage check")
			return
		}

		captureTime := time.Now()
		totalCost := 0.0
		asks := make([]struct {
			Outcome string
			Price   float64
			Size    float64
		}, 0, len(market.Outcomes))

		// Collect ask prices for NO shares
		// market.Outcomes contains only NO share token IDs (extracted by SDK's FetchMarketOutcomes)
		emptyBooks := []string{}
		for _, noTokenID := range market.Outcomes {
			data, exists := cache.Get(noTokenID)
			if !exists || data.BestAsk == 0 {
				emptyBooks = append(emptyBooks, noTokenID)
				continue // Skip this one but continue checking others
			}
			asks = append(asks, struct {
				Outcome string
				Price   float64
				Size    float64
			}{noTokenID, data.BestAsk, data.BestAskSize})
			totalCost += data.BestAsk
		}

		// If we have some empty books, double-check them
		if len(emptyBooks) > 0 {
			for _, tokenID := range emptyBooks {
				book, err := bot.client.GetOrderBook(tokenID)
				if err == nil && len(book.Asks) > 0 {
					askPrice, _ := strconv.ParseFloat(book.Asks[0].Price, 64)
					askSize, _ := strconv.ParseFloat(book.Asks[0].Size, 64)
					if askPrice > 0 {
						asks = append(asks, struct {
							Outcome string
							Price   float64
							Size    float64
						}{tokenID, askPrice, askSize})
						totalCost += askPrice
					}
				}
			}
		}

		// Skip if we have too few liquid outcomes
		if len(asks) < 2 {
			return // Need at least 2 NO shares for arbitrage
		}

		// Check for negative risk opportunity with available shares
		// If we buy k NO shares, we're guaranteed k-1 payouts (since at most 1 can lose)
		// So arbitrage exists if: totalCost < k-1
		k := float64(len(asks))
		if totalCost < (k-1)-cfg.MinEdge && totalCost <= bot.config.MaxSpendUSDC {
			atomic.AddInt64(&bot.stats.opportunitiesFound, 1)
			log.Printf("[TAKER] Negative risk arbitrage detected! Market: %s", market.Name)
			log.Printf("[TAKER] Buying %d liquid NO shares (out of %d total)", len(asks), len(market.Outcomes))
			log.Printf("[TAKER] Total cost: $%.4f < $%.4f (guaranteed payout), edge: $%.4f",
				totalCost, k-1, (k-1)-totalCost)
			log.Printf("[TAKER] Within budget limit of $%.2f", bot.config.MaxSpendUSDC)

			// Execute trades concurrently for speed
			var wg sync.WaitGroup
			for _, ask := range asks {
				wg.Add(1)
				go func(outcome string, price, size float64) {
					defer wg.Done()
					orderStartTime := time.Now()
					bot.metrics.RecordCaptureToOrder(orderStartTime.Sub(captureTime))
					bot.placeMarketBuy(outcome, price, size)
				}(ask.Outcome, ask.Price, ask.Size)
			}
			wg.Wait()

			// Brief cooldown after execution
			time.Sleep(time.Second)
		} else if totalCost > bot.config.MaxSpendUSDC && totalCost < (k-1)-cfg.MinEdge {
			log.Printf("[TAKER] Arbitrage opportunity skipped - cost $%.4f exceeds budget $%.2f",
				totalCost, bot.config.MaxSpendUSDC)
		}
	}

	for {
		select {
		case <-bot.ctx.Done():
			return
		case <-ticker.C:
			checkArbitrage()
		case <-bot.arbTrigger:
			// Fast path: token already validated in WebSocket handler
			// Execute immediately without additional checks
			go checkArbitrage()
		}
	}
}

func (bot *ArbBot) startMakerTakerStrategy(cache *MarketCache, cfg StrategyConfig) {
	for _, market := range bot.config.Markets {
		for i := 0; i < cfg.WorkerAmount; i++ {
			bot.wg.Add(1)
			go bot.runMakerTakerArb(market, cache, cfg)
		}
	}
}

func (bot *ArbBot) runMakerTakerArb(market MarketGroup, cache *MarketCache, cfg StrategyConfig) {
	defer bot.wg.Done()

	n := float64(len(market.Outcomes))
	ticker := time.NewTicker(time.Duration(cfg.PollInterval) * time.Millisecond)
	defer ticker.Stop()

	// Initialize active bids map for this market
	bot.bidsMutex.Lock()
	if bot.activeBids[market.Name] == nil {
		bot.activeBids[market.Name] = make(map[string]*ActiveBid)
	}
	bot.bidsMutex.Unlock()

	for {
		select {
		case <-bot.ctx.Done():
			return
		case <-ticker.C:
			captureTime := time.Now()

			// Get best asks for all outcomes
			bestAsks := make(map[string]float64)
			validData := true

			for _, outcome := range market.Outcomes {
				data, exists := cache.Get(outcome)
				if !exists || data.BestAsk == 0 {
					validData = false
					break
				}
				bestAsks[outcome] = data.BestAsk
			}

			if !validData {
				continue
			}

			// Update bids for each outcome
			for _, outcomeJ := range market.Outcomes {
				sumOtherAsks := 0.0
				for _, outcomeK := range market.Outcomes {
					if outcomeK != outcomeJ {
						sumOtherAsks += bestAsks[outcomeK]
					}
				}

				desiredBid := (n - 1 - cfg.ExtraEdge) - sumOtherAsks

				if desiredBid > 0.01 { // Minimum tick size
					bot.updateBid(market.Name, outcomeJ, desiredBid)
				} else {
					bot.cancelBid(market.Name, outcomeJ)
				}
			}

			// Check for filled orders
			filledOutcome := bot.checkForFilledBid(market.Name)
			if filledOutcome != "" {
				atomic.AddInt64(&bot.stats.opportunitiesFound, 1)
				log.Printf("[MAKER-TAKER] NO share bid filled for outcome %s! Sweeping remaining NO shares", filledOutcome)

				// Sweep remaining outcomes
				var wg sync.WaitGroup
				for _, outcome := range market.Outcomes {
					if outcome != filledOutcome {
						data, _ := cache.Get(outcome)
						wg.Add(1)
						go func(o string, price, size float64) {
							defer wg.Done()
							orderStartTime := time.Now()
							bot.metrics.RecordCaptureToOrder(orderStartTime.Sub(captureTime))
							bot.placeMarketBuy(o, price, size)
						}(outcome, data.BestAsk, data.BestAskSize)
					}
				}
				wg.Wait()

				// Clear active bids for this market
				bot.clearMarketBids(market.Name)

				// Brief cooldown
				time.Sleep(time.Second)
			}
		}
	}
}

func (bot *ArbBot) updateBid(marketName, outcome string, price float64) {
	bot.bidsMutex.Lock()
	defer bot.bidsMutex.Unlock()

	currentBid := bot.activeBids[marketName][outcome]
	if currentBid == nil || currentBid.Price != price {
		// Cancel existing bid if any
		if currentBid != nil {
			_, err := bot.client.Cancel(currentBid.OrderID)
			if err != nil {
				log.Printf("Error canceling old bid: %v", err)
			}
		}

		// Place new bid
		orderArgs := &types.OrderArgs{
			TokenID:    outcome,
			Price:      price,
			Size:       100, // Adjust size as needed
			Side:       "BUY",
			FeeRateBps: 0,
			Nonce:      0,
			Expiration: time.Now().Add(24 * time.Hour).Unix(),
			Taker:      "0x0000000000000000000000000000000000000000",
		}

		order, err := bot.client.CreateOrder(orderArgs, nil)
		if err != nil {
			log.Printf("Error creating bid: %v", err)
			return
		}

		resp, err := bot.client.PostOrder(order, types.OrderTypeGTC)
		if err != nil {
			log.Printf("Error posting bid: %v", err)
			return
		}

		if orderID, exists := resp["orderID"]; exists {
			bot.activeBids[marketName][outcome] = &ActiveBid{
				OrderID: fmt.Sprintf("%v", orderID),
				Price:   price,
				Size:    100,
			}
		}
	}
}

func (bot *ArbBot) cancelBid(marketName, outcome string) {
	bot.bidsMutex.Lock()
	defer bot.bidsMutex.Unlock()

	if bid := bot.activeBids[marketName][outcome]; bid != nil {
		_, err := bot.client.Cancel(bid.OrderID)
		if err != nil {
			log.Printf("Error canceling bid: %v", err)
		}
		delete(bot.activeBids[marketName], outcome)
	}
}

func (bot *ArbBot) checkForFilledBid(marketName string) string {
	bot.bidsMutex.RLock()
	defer bot.bidsMutex.RUnlock()

	for outcome, bid := range bot.activeBids[marketName] {
		// Check order status
		order, err := bot.client.GetOrder(bid.OrderID)
		if err != nil {
			continue
		}

		// Check if order is filled
		if status, exists := order["status"]; exists && status == "FILLED" {
			return outcome
		}
	}
	return ""
}

func (bot *ArbBot) clearMarketBids(marketName string) {
	bot.bidsMutex.Lock()
	defer bot.bidsMutex.Unlock()

	for outcome, bid := range bot.activeBids[marketName] {
		_, err := bot.client.Cancel(bid.OrderID)
		if err != nil {
			log.Printf("Error canceling bid: %v", err)
		}
		delete(bot.activeBids[marketName], outcome)
	}
}

func (bot *ArbBot) placeMarketBuy(tokenID string, price, size float64) {
	startTime := time.Now()
	atomic.AddInt64(&bot.stats.ordersPlaced, 1)

	// For market orders, use a price slightly above best ask to ensure fill
	marketPrice := price * 1.01 // 1% above best ask

	marketOrderArgs := &types.MarketOrderArgs{
		TokenID:    tokenID,
		Amount:     size * marketPrice, // For BUY orders, amount is in dollars
		Side:       "BUY",
		Price:      marketPrice,
		FeeRateBps: 0,
		Nonce:      0,
		Taker:      "0x0000000000000000000000000000000000000000",
	}

	order, err := bot.client.CreateMarketOrder(marketOrderArgs, nil)
	if err != nil {
		atomic.AddInt64(&bot.stats.ordersFailed, 1)
		log.Printf("Error creating market buy for %s: %v", tokenID, err)
		return
	}

	_, err = bot.client.PostOrder(order, types.OrderTypeFOK)

	executionTime := time.Since(startTime)
	bot.metrics.RecordOrderExecution(executionTime)

	if err != nil {
		atomic.AddInt64(&bot.stats.ordersFailed, 1)
		log.Printf("Error placing market buy for %s: %v (execution time: %v)", tokenID, err, executionTime)
	} else {
		atomic.AddInt64(&bot.stats.ordersSucceeded, 1)
	}
}

func saveUpdatedConfig(cfg BotConfig) error {
	configData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("arbitrage_config.json", configData, 0644)
}
