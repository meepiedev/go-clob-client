package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/polymarket/go-order-utils/pkg/model"
	"github.com/pooofdevelopment/go-clob-client/pkg/client"
	"github.com/pooofdevelopment/go-clob-client/pkg/types"
	"github.com/pooofdevelopment/go-clob-client/pkg/websocket"
)

// Constants
const (
	maxRequestsPerSecond = 5
	requestInterval      = 200 * time.Millisecond // Fixed at 5 req/s
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
	DryRun        bool                      `json:"dry_run"` // If true, don't place real orders
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
	ExtraEdge    float64 `json:"extra_edge,omitempty"` // For maker-taker only
}

// MarketData holds orderbook data
type MarketData struct {
	TokenID     string
	OutcomeName string
	BestAsk     float64
	BestAskSize float64
	BestBid     float64
	BestBidSize float64
	LastUpdate  time.Time
}

// ArbitrageOpportunity represents a detected arbitrage opportunity
type ArbitrageOpportunity struct {
	MarketName       string
	TotalCost        float64
	GuaranteedPayout float64
	Edge             float64
	EdgePercent      float64
	NumShares        int
	Outcomes         []MarketData
	DetectedAt       time.Time
	WouldExecute     bool
}

// Stats tracks arbitrage bot performance
type Stats struct {
	opportunitiesFound int64
	ordersPlaced       int64
	ordersSucceeded    int64
	ordersFailed       int64
	totalPnL           float64
	startTime          time.Time
}

// ViewMode represents different UI views
type ViewMode int

const (
	ViewOverview ViewMode = iota
	ViewOrderbook
)

// Model represents the Tea application state
type Model struct {
	client        *client.ClobClient
	config        BotConfig
	markets       []MarketGroup
	marketData    sync.Map // tokenID -> *MarketData (lock-free concurrent access)
	opportunities []ArbitrageOpportunity
	stats         *Stats

	// UI state
	marketTables    map[string]table.Model
	logMessages     []string
	width           int
	height          int
	selectedEvent   int // Which event (MarketGroup) is selected
	selectedOutcome int // Which outcome within the selected event
	viewMode        ViewMode

	// Orderbook view state
	orderbookEvent   *MarketGroup            // The event whose orderbook we're viewing
	orderbookTokenID string                  // The specific token ID we're viewing
	orderbookData    *types.OrderBookSummary // Full orderbook from WebSocket
	wsClient         *websocket.Client       // WebSocket client for orderbook

	// Polling state
	lastPollTime   time.Time
	lastPollStatus int
	pollInProgress bool

	// Order tracking to prevent duplicates
	activeOrders sync.Map // marketName -> bool (true if has active orders)

	// Control
	ctx    context.Context
	cancel context.CancelFunc
}

// Message types for Tea
type tickMsg time.Time
type marketDataMsg struct {
	data       []MarketData
	pollTime   time.Time
	statusCode int
}
type logMsg string
type orderbookUpdateMsg struct {
	tokenID string
	book    *types.OrderBookSummary
}

// Styles
var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("63")).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	opportunityStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("46")).
				Bold(true)

	priceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	buyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)

	sellStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	spreadStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Bold(true)
)

func main() {
	// Set up logging to file
	logFile, err := os.OpenFile("arbitrage_bot_polling.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// Load configuration
	configData, err := os.ReadFile("arbitrage_config_polling.json")
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	var cfg BotConfig
	if err := json.Unmarshal(configData, &cfg); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	// Create client
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
		// First create client with private key only
		var walletAddr *string
		if strings.ToLower(cfg.SignatureType) == "poly_proxy" && cfg.Address != "" {
			walletAddr = &cfg.Address
		}
		clobClient, err = client.NewClobClient(host, chainID, cfg.PrivateKey, nil, sigType, walletAddr)
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

		// Save the generated credentials back to config
		cfg.APIKey = creds.ApiKey
		cfg.APISecret = creds.ApiSecret
		cfg.APIPassphrase = creds.ApiPassphrase

	} else if cfg.APIKey != "" {
		// Create L2 client directly with provided credentials
		log.Println("Using provided API credentials")
		creds := &types.ApiCreds{
			ApiKey:        cfg.APIKey,
			ApiSecret:     cfg.APISecret,
			ApiPassphrase: cfg.APIPassphrase,
		}

		var walletAddr *string
		if strings.ToLower(cfg.SignatureType) == "poly_proxy" && cfg.Address != "" {
			walletAddr = &cfg.Address
		}
		clobClient, err = client.NewClobClient(host, chainID, cfg.PrivateKey, creds, sigType, walletAddr)
		if err != nil {
			log.Fatalf("Failed to create L2 client: %v", err)
		}
	} else {
		log.Fatalf("Either API credentials or private key must be provided")
	}

	// Check markets and fetch outcomes
	tempClient, _ := client.NewClobClient(host, chainID, "", nil, nil, nil)
	for i, market := range cfg.Markets {
		if market.Slug != "" {
			// Check if market is negative risk
			isNegRisk, err := tempClient.CheckNegativeRisk(market.Slug)
			if err != nil {
				log.Printf("Warning: Failed to check negRisk status: %v", err)
			} else if !isNegRisk {
				log.Fatalf("ERROR: Market '%s' is NOT a negative risk market!", market.Name)
			}

			// Fetch outcomes if not already provided
			if len(market.Outcomes) == 0 {
				outcomes, outcomeNames, err := tempClient.FetchMarketOutcomes(market.Slug)
				if err != nil {
					log.Fatalf("Failed to fetch outcomes for %s: %v", market.Name, err)
				}
				cfg.Markets[i].Outcomes = outcomes
				cfg.Markets[i].OutcomeNames = outcomeNames
			}
		}
	}

	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	// Order builder is not needed - client handles order creation internally

	// Initialize model
	m := Model{
		client:       clobClient,
		config:       cfg,
		markets:      cfg.Markets,
		marketTables: make(map[string]table.Model),
		stats: &Stats{
			startTime: time.Now(),
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize market tables
	for _, market := range m.markets {
		columns := []table.Column{
			{Title: "Outcome", Width: 30},
			{Title: "Best Bid", Width: 15},
			{Title: "Bid Size", Width: 12},
			{Title: "Best Ask", Width: 15},
			{Title: "Ask Size", Width: 12},
			{Title: "Updated", Width: 12},
		}

		t := table.New(
			table.WithColumns(columns),
			table.WithFocused(false),
			table.WithHeight(len(market.Outcomes)+1),
		)

		s := table.DefaultStyles()
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			BorderBottom(true).
			Bold(false)
		// Remove table selection styling since we handle it ourselves
		s.Selected = s.Selected.
			Foreground(lipgloss.NoColor{}).
			Background(lipgloss.NoColor{}).
			Bold(false)
		t.SetStyles(s)

		m.marketTables[market.Name] = t
	}

	// Polling will be started by Init()

	// Create Tea program
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		m.startPolling(),
		tea.EnterAltScreen,
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.viewMode {
		case ViewOverview:
			switch msg.String() {
			case "q", "ctrl+c":
				m.cancel()
				return m, tea.Quit
			case "up", "k":
				if m.selectedOutcome > 0 {
					m.selectedOutcome--
				} else if m.selectedEvent > 0 {
					// Move to previous event's last outcome
					m.selectedEvent--
					m.selectedOutcome = len(m.markets[m.selectedEvent].Outcomes) - 1
				}
			case "down", "j":
				if m.selectedEvent < len(m.markets) &&
					m.selectedOutcome < len(m.markets[m.selectedEvent].Outcomes)-1 {
					m.selectedOutcome++
				} else if m.selectedEvent < len(m.markets)-1 {
					// Move to next event's first outcome
					m.selectedEvent++
					m.selectedOutcome = 0
				}
			case "left", "h":
				// Switch to previous event
				if m.selectedEvent > 0 {
					m.selectedEvent--
					m.selectedOutcome = 0
				}
			case "right", "l":
				// Switch to next event
				if m.selectedEvent < len(m.markets)-1 {
					m.selectedEvent++
					m.selectedOutcome = 0
				}
			case "enter":
				// Switch to orderbook view for selected outcome
				if m.selectedEvent < len(m.markets) &&
					m.selectedOutcome < len(m.markets[m.selectedEvent].Outcomes) {
					m.viewMode = ViewOrderbook
					m.orderbookEvent = &m.markets[m.selectedEvent]
					m.orderbookTokenID = m.markets[m.selectedEvent].Outcomes[m.selectedOutcome]
					// Subscribe to WebSocket for this token
					return m, m.subscribeToOrderbook()
				}
			}
		case ViewOrderbook:
			switch msg.String() {
			case "q", "esc":
				// Return to overview
				m.viewMode = ViewOverview
				// Unsubscribe from WebSocket
				if m.wsClient != nil {
					m.wsClient.Close()
					m.wsClient = nil
				}
				m.orderbookData = nil
			case "up", "k":
				// In orderbook view, could scroll through order levels if we had full depth
				// For now, just a placeholder
			case "down", "j":
				// In orderbook view, could scroll through order levels if we had full depth
				// For now, just a placeholder
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		// Update UI periodically
		return m, tickCmd()

	case marketDataMsg:
		// Update polling status
		m.lastPollTime = msg.pollTime
		m.lastPollStatus = msg.statusCode
		m.pollInProgress = false

		// Update market data
		for _, data := range msg.data {
			m.marketData.Store(data.TokenID, &data)
		}
		// Check for arbitrage opportunities
		opportunities := m.checkArbitrage()
		for _, opp := range opportunities {
			// Add to opportunities list
			m.opportunities = append(m.opportunities, opp)
			if len(m.opportunities) > 10 {
				m.opportunities = m.opportunities[len(m.opportunities)-10:]
			}
			atomic.AddInt64(&m.stats.opportunitiesFound, 1)
		}

		// Continue polling
		return m, m.startPolling()

	case logMsg:
		msgStr := string(msg)
		// Filter out noisy WebSocket messages from UI
		if msgStr != "Connected to orderbook WebSocket" &&
			!strings.Contains(msgStr, "PONG") &&
			!strings.Contains(msgStr, "ping") {
			m.logMessages = append(m.logMessages, msgStr)
			// Keep only last 10 messages in memory (we'll display last 5)
			if len(m.logMessages) > 10 {
				m.logMessages = m.logMessages[len(m.logMessages)-10:]
			}
		}
		// Always log to file
		log.Println(msgStr)

	case orderbookUpdateMsg:
		// Update orderbook data if it's for the current token
		if msg.tokenID == m.orderbookTokenID {
			m.orderbookData = msg.book
		}
		// Continue listening for more updates if in orderbook view
		if m.viewMode == ViewOrderbook {
			return m, waitForOrderbookUpdate()
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	switch m.viewMode {
	case ViewOrderbook:
		return m.viewOrderbook()
	default:
		return m.viewOverview()
	}
}

func (m Model) viewOverview() string {

	// Header
	header := headerStyle.Width(m.width).Render("🎯 Negative Risk Arbitrage Bot (Polling Mode)")

	// Stats
	runtime := time.Since(m.stats.startTime)

	// Polling status
	pollStatus := "Never polled"
	if !m.lastPollTime.IsZero() {
		timeSincePoll := time.Since(m.lastPollTime)
		statusIcon := "✓"
		if m.lastPollStatus != 200 && m.lastPollStatus != 0 {
			statusIcon = "✗"
		}
		pollStatus = fmt.Sprintf("Last poll: %s ago %s (HTTP %d)",
			timeSincePoll.Round(time.Millisecond),
			statusIcon,
			m.lastPollStatus)
	}

	statsText := fmt.Sprintf(
		"Runtime: %v | Opportunities: %d | Orders: %d/%d | Mode: %s | %s",
		runtime.Round(time.Second),
		atomic.LoadInt64(&m.stats.opportunitiesFound),
		atomic.LoadInt64(&m.stats.ordersSucceeded),
		atomic.LoadInt64(&m.stats.ordersPlaced),
		func() string {
			if m.config.DryRun {
				return "DRY RUN"
			} else {
				return "LIVE"
			}
		}(),
		pollStatus,
	)
	stats := dimStyle.Render(statsText)

	// Event tabs - show all events as tabs
	var eventTabs []string
	for i, event := range m.markets {
		var tabStyle lipgloss.Style
		if i == m.selectedEvent {
			tabStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("229")).
				Background(lipgloss.Color("57")).
				Padding(0, 2)
		} else {
			tabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Background(lipgloss.Color("237")).
				Padding(0, 2)
		}
		eventTabs = append(eventTabs, tabStyle.Render(event.Name))
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, eventTabs...)

	// Display only the selected event's market data
	var marketSection string
	if m.selectedEvent < len(m.markets) {
		event := m.markets[m.selectedEvent]
		// Calculate arbitrage for this event
		totalCost := 0.0
		liquidCount := 0
		var rows []table.Row

		for outcomeIdx, tokenID := range event.Outcomes {
			outcomeName := "Unknown"
			if name, exists := event.OutcomeNames[tokenID]; exists {
				outcomeName = name
				if len(outcomeName) > 28 {
					outcomeName = outcomeName[:25] + "..."
				}
			}

			// Add selection indicator to outcome name
			displayName := outcomeName
			if outcomeIdx == m.selectedOutcome {
				displayName = "▶ " + outcomeName
			} else {
				displayName = "  " + outcomeName
			}

			row := table.Row{
				displayName,
				"-",
				"-",
				"-",
				"-",
				"Never",
			}

			if data, exists := m.marketData.Load(tokenID); exists {
				md := data.(*MarketData)
				if md.BestBid > 0 {
					row[1] = fmt.Sprintf("$%.4f", md.BestBid)
					row[2] = fmt.Sprintf("%.1f", md.BestBidSize)
				}
				if md.BestAsk > 0 {
					row[3] = fmt.Sprintf("$%.4f", md.BestAsk)
					row[4] = fmt.Sprintf("%.1f", md.BestAskSize)
					totalCost += md.BestAsk
					liquidCount++
				}
				row[5] = time.Since(md.LastUpdate).Round(time.Second).String()
			}

			rows = append(rows, row)
		}

		// Update table
		if t, exists := m.marketTables[event.Name]; exists {
			t.SetRows(rows)

			// Event header with arbitrage calculation
			eventHeader := titleStyle.Render(event.Name)

			if liquidCount >= 2 {
				k := float64(liquidCount)
				edge := (k - 1) - totalCost
				edgePercent := (edge / (k - 1)) * 100

				arbInfo := fmt.Sprintf(
					"Total Cost: $%.4f | Guaranteed: $%.4f | Edge: $%.4f (%.2f%%)",
					totalCost, k-1, edge, edgePercent,
				)

				if edge > 0.01 { // Opportunity threshold
					arbInfo = opportunityStyle.Render("🚨 " + arbInfo)
				} else {
					arbInfo = dimStyle.Render(arbInfo)
				}

				eventHeader = fmt.Sprintf("%s\n%s", eventHeader, arbInfo)
			} else {
				eventHeader = fmt.Sprintf("%s\n%s",
					eventHeader,
					dimStyle.Render(fmt.Sprintf("Waiting for liquidity (%d/%d outcomes)", liquidCount, len(event.Outcomes))),
				)
			}

			marketSection = baseStyle.Render(
				lipgloss.JoinVertical(lipgloss.Left,
					eventHeader,
					t.View(),
				),
			)
		}
	}

	// Recent opportunities
	oppHeader := titleStyle.Render("Recent Opportunities")
	var oppLines []string
	for i := len(m.opportunities) - 1; i >= 0 && len(oppLines) < 5; i-- {
		opp := m.opportunities[i]
		status := "🟡 DETECTED"
		if opp.WouldExecute {
			status = "🟢 WOULD EXECUTE"
		}

		// Truncate market name if too long
		marketName := opp.MarketName
		if len(marketName) > 25 {
			marketName = marketName[:22] + "..."
		}

		oppLine := fmt.Sprintf(
			"%s %s - %s: $%.2f (%.1f%%) %d shares",
			status,
			opp.DetectedAt.Format("15:04:05"),
			marketName,
			opp.Edge,
			opp.EdgePercent,
			opp.NumShares,
		)
		oppLines = append(oppLines, oppLine)
	}

	if len(oppLines) == 0 {
		oppLines = append(oppLines, dimStyle.Render("No opportunities detected yet..."))
	}

	oppSection := baseStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			oppHeader,
			lipgloss.JoinVertical(lipgloss.Left, oppLines...),
		),
	)

	// Log messages - show last 5
	logHeader := titleStyle.Render("System Log")
	var displayLogs []string
	if len(m.logMessages) > 0 {
		// Take only the last 5 logs
		start := len(m.logMessages) - 5
		if start < 0 {
			start = 0
		}
		displayLogs = m.logMessages[start:]
	}
	if len(displayLogs) == 0 {
		displayLogs = []string{dimStyle.Render("No recent log messages...")}
	}
	logSection := baseStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			logHeader,
			lipgloss.JoinVertical(lipgloss.Left, displayLogs...),
		),
	)

	// Compose final view
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		stats,
		"",
		tabBar,
		"",
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			marketSection,
			"  ",
			lipgloss.JoinVertical(
				lipgloss.Left,
				oppSection,
				"",
				logSection,
			),
		),
	)

	// Footer
	footer := dimStyle.Render("↑/↓ Navigate Markets | ←/→ Switch Events | Enter: View Orderbook | 'q' to quit | Logs: arbitrage_bot_polling.log")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		"",
		footer,
	)
}

func (m *Model) startPolling() tea.Cmd {
	return func() tea.Msg {
		ticker := time.NewTicker(requestInterval)
		defer ticker.Stop()

		for {
			select {
			case <-m.ctx.Done():
				return nil
			case <-ticker.C:
				pollTime := time.Now()
				updates, statusCode := m.pollMarketData()
				return marketDataMsg{
					data:       updates,
					pollTime:   pollTime,
					statusCode: statusCode,
				}
			}
		}
	}
}

func (m *Model) pollMarketData() ([]MarketData, int) {
	// Collect all token IDs grouped by market for immediate arbitrage checking
	tokensByMarket := make(map[string][]string)
	var bookParams []types.BookParams

	for _, market := range m.markets {
		tokensByMarket[market.Name] = market.Outcomes
		for _, tokenID := range market.Outcomes {
			bookParams = append(bookParams, types.BookParams{
				TokenID: tokenID,
			})
		}
	}

	// Fetch all orderbooks in one request
	orderbooks, err := m.client.GetOrderBooks(bookParams)
	if err != nil {
		// Log error but continue
		log.Printf("Error fetching orderbooks: %v", err)
		return nil, 0
	}

	// Process orderbooks concurrently for faster updates
	var wg sync.WaitGroup
	updateChan := make(chan MarketData, len(orderbooks))

	for _, book := range orderbooks {
		wg.Add(1)
		go func(b types.OrderBookSummary) {
			defer wg.Done()
			data := MarketData{
				TokenID:    b.AssetID,
				LastUpdate: time.Now(),
			}

			// Get outcome name
			for _, market := range m.markets {
				if name, exists := market.OutcomeNames[b.AssetID]; exists {
					data.OutcomeName = name
					break
				}
			}

			// Find best ask (lowest ask price)
			if len(b.Asks) > 0 {
				bestAskPrice := math.MaxFloat64
				bestAskSize := 0.0

				for _, ask := range b.Asks {
					price, err := strconv.ParseFloat(ask.Price, 64)
					if err != nil {
						continue
					}
					size, _ := strconv.ParseFloat(ask.Size, 64)

					if price < bestAskPrice && price > 0 {
						bestAskPrice = price
						bestAskSize = size
					}
				}

				if bestAskPrice < math.MaxFloat64 {
					data.BestAsk = bestAskPrice
					data.BestAskSize = bestAskSize
				}
			}

			// Find best bid (highest bid price)
			if len(b.Bids) > 0 {
				bestBidPrice := 0.0
				bestBidSize := 0.0

				for _, bid := range b.Bids {
					price, err := strconv.ParseFloat(bid.Price, 64)
					if err != nil {
						continue
					}
					size, _ := strconv.ParseFloat(bid.Size, 64)

					if price > bestBidPrice {
						bestBidPrice = price
						bestBidSize = size
					}
				}

				if bestBidPrice > 0 {
					data.BestBid = bestBidPrice
					data.BestBidSize = bestBidSize
				}
			}

			// Store in lock-free sync.Map immediately
			m.marketData.Store(b.AssetID, &data)
			updateChan <- data
		}(book)
	}

	// Close channel when all done
	go func() {
		wg.Wait()
		close(updateChan)
	}()

	// Collect updates
	var updates []MarketData
	for update := range updateChan {
		updates = append(updates, update)
	}

	// Immediately check for arbitrage opportunities after updating data
	go m.checkAllMarketsForArbitrage()

	// Return 200 for successful response
	return updates, 200
}

func (m *Model) checkArbitrage() []ArbitrageOpportunity {
	var opportunities []ArbitrageOpportunity

	// Check each market for arbitrage opportunities
	for _, market := range m.markets {
		totalCost := 0.0
		liquidCount := 0
		var outcomes []MarketData

		for _, tokenID := range market.Outcomes {
			if data, exists := m.marketData.Load(tokenID); exists {
				md := data.(*MarketData)
				if md.BestAsk > 0 {
					totalCost += md.BestAsk
					liquidCount++
					outcomes = append(outcomes, *md)
				}
			}
		}

		// Need at least 2 liquid outcomes for arbitrage
		if liquidCount < 2 {
			continue
		}

		// Calculate arbitrage
		k := float64(liquidCount)
		guaranteedPayout := k - 1
		edge := guaranteedPayout - totalCost
		edgePercent := (edge / guaranteedPayout) * 100

		// Check if this is an opportunity based on strategy settings
		minEdge := 0.001 // Default min edge
		if takerCfg, exists := m.config.Strategies["taker"]; exists && takerCfg.Enabled {
			minEdge = takerCfg.MinEdge
		}

		if edge > minEdge {
			opp := ArbitrageOpportunity{
				MarketName:       market.Name,
				TotalCost:        totalCost,
				GuaranteedPayout: guaranteedPayout,
				Edge:             edge,
				EdgePercent:      edgePercent,
				NumShares:        liquidCount,
				Outcomes:         outcomes,
				DetectedAt:       time.Now(),
				WouldExecute:     edge > minEdge && totalCost <= m.config.MaxSpendUSDC,
			}

			opportunities = append(opportunities, opp)

			// Log opportunity details
			if opp.WouldExecute {
				// In dry run mode, just log what we would do
				if m.config.DryRun {
					// Simulate order placement
					atomic.AddInt64(&m.stats.ordersPlaced, int64(opp.NumShares))
					atomic.AddInt64(&m.stats.ordersSucceeded, int64(opp.NumShares))
				} else {
					// Place real orders
					for range outcomes {
						// Here you would place the actual market buy order
						// bot.placeMarketBuy(outcome.TokenID, outcome.BestAsk, outcome.BestAskSize)
					}
				}
			}
		}
	}

	return opportunities
}

// Check all markets for arbitrage opportunities (called immediately after data update)
func (m *Model) checkAllMarketsForArbitrage() {
	for _, market := range m.markets {
		opportunities := m.checkArbitrageForMarket(market)
		for _, opp := range opportunities {
			if opp.WouldExecute && !m.config.DryRun {
				// Execute arbitrage with parallel orders
				go m.executeArbitrageParallel(opp, market)
			}
		}
	}
}

// Check arbitrage for a specific market
func (m *Model) checkArbitrageForMarket(market MarketGroup) []ArbitrageOpportunity {
	var opportunities []ArbitrageOpportunity

	totalCost := 0.0
	liquidCount := 0
	var outcomes []MarketData

	for _, tokenID := range market.Outcomes {
		if data, exists := m.marketData.Load(tokenID); exists {
			md := data.(*MarketData)
			if md.BestAsk > 0 {
				totalCost += md.BestAsk
				liquidCount++
				outcomes = append(outcomes, *md)
			}
		}
	}

	// Need at least 2 liquid outcomes for arbitrage
	if liquidCount < 2 {
		return opportunities
	}

	// Calculate arbitrage
	k := float64(liquidCount)
	guaranteedPayout := k - 1
	edge := guaranteedPayout - totalCost
	edgePercent := (edge / guaranteedPayout) * 100

	// Check if this is an opportunity based on strategy settings
	minEdge := 0.001 // Default min edge
	if takerCfg, exists := m.config.Strategies["taker"]; exists && takerCfg.Enabled {
		minEdge = takerCfg.MinEdge
	}

	if edge > minEdge {
		opp := ArbitrageOpportunity{
			MarketName:       market.Name,
			TotalCost:        totalCost,
			GuaranteedPayout: guaranteedPayout,
			Edge:             edge,
			EdgePercent:      edgePercent,
			NumShares:        liquidCount,
			Outcomes:         outcomes,
			DetectedAt:       time.Now(),
			WouldExecute:     edge > minEdge && totalCost <= m.config.MaxSpendUSDC,
		}

		// Add to opportunities for UI display
		m.opportunities = append(m.opportunities, opp)
		if len(m.opportunities) > 10 {
			m.opportunities = m.opportunities[len(m.opportunities)-10:]
		}
		atomic.AddInt64(&m.stats.opportunitiesFound, 1)

		opportunities = append(opportunities, opp)

		// Log opportunity
		if opp.WouldExecute {
			log.Printf("🚨 ARBITRAGE OPPORTUNITY: %s - Edge: $%.4f (%.2f%%) - Cost: $%.4f",
				market.Name, edge, edgePercent, totalCost)
		}
	}

	return opportunities
}

// Execute arbitrage with parallel order placement
func (m *Model) executeArbitrageParallel(opp ArbitrageOpportunity, market MarketGroup) {
	// Check if we already have active orders for this market
	if hasActive, exists := m.activeOrders.Load(market.Name); exists && hasActive.(bool) {
		log.Printf("Skipping arbitrage for %s - orders still active", market.Name)
		return
	}

	log.Printf("Executing arbitrage for %s - %d orders", market.Name, len(opp.Outcomes))

	// Mark this market as having active orders
	m.activeOrders.Store(market.Name, true)

	// Create order results channel
	results := make(chan OrderResult, len(opp.Outcomes))
	var wg sync.WaitGroup

	// Place all orders in parallel
	for i, outcome := range opp.Outcomes {
		wg.Add(1)
		go func(idx int, md MarketData) {
			defer wg.Done()

			// Find the token ID for this outcome
			tokenID := ""
			for _, tid := range market.Outcomes {
				if data, exists := m.marketData.Load(tid); exists {
					if data.(*MarketData).OutcomeName == md.OutcomeName {
						tokenID = tid
						break
					}
				}
			}

			if tokenID == "" {
				results <- OrderResult{
					Success: false,
					Error:   fmt.Errorf("could not find token ID for outcome %s", md.OutcomeName),
				}
				return
			}

			// Create limit order with best ask price and 2-second expiration
			orderPrice := md.BestAsk // Use best ask price directly
			shares := 1.0            // Buy 1 share

			// Ensure minimum order size of $1.00
			if shares*orderPrice < 1.0 {
				shares = 1.0 / orderPrice
				shares = float64(int(shares + 0.99)) // Round up
			}

			// Set expiration to 62 seconds from now
			expiration := time.Now().Add(62 * time.Second).Unix()

			orderArgs := &types.OrderArgs{
				TokenID:    tokenID,
				Price:      orderPrice,
				Size:       shares,
				Side:       types.BUY,
				FeeRateBps: 0,
				Nonce:      0,
				Expiration: expiration,
				Taker:      types.ZeroAddress,
			}

			// Create options with negRisk flag (always true since we check at startup)
			negRisk := true
			orderOptions := &types.PartialCreateOrderOptions{
				NegRisk: &negRisk,
			}

			// Create and sign order
			signedOrder, err := m.client.CreateOrder(orderArgs, orderOptions)
			if err != nil {
				results <- OrderResult{
					TokenID: tokenID,
					Success: false,
					Error:   err,
				}
				return
			}

			// Post order
			resp, err := m.client.PostOrder(signedOrder, types.OrderTypeGTD)
			if err != nil {
				results <- OrderResult{
					TokenID: tokenID,
					Success: false,
					Error:   err,
				}
				return
			}

			// Extract order ID from response
			orderID := ""
			if id, ok := resp["orderID"].(string); ok {
				orderID = id
			}

			results <- OrderResult{
				TokenID: tokenID,
				OrderID: orderID,
				Success: true,
			}

			atomic.AddInt64(&m.stats.ordersPlaced, 1)
			atomic.AddInt64(&m.stats.ordersSucceeded, 1)

		}(i, outcome)
	}

	// Wait for all orders to complete
	go func() {
		wg.Wait()
		close(results)

		// Collect results
		successCount := 0
		for result := range results {
			if result.Success {
				successCount++
				log.Printf("✅ Order placed: Token %s, Order ID: %s", result.TokenID, result.OrderID)
			} else {
				log.Printf("❌ Order failed: Token %s, Error: %v", result.TokenID, result.Error)
				atomic.AddInt64(&m.stats.ordersFailed, 1)
			}
		}

		log.Printf("Arbitrage execution complete: %d/%d orders successful", successCount, len(opp.Outcomes))

		// Clear the active order status regardless of success/failure
		// This allows retry on failure
		m.activeOrders.Delete(market.Name)
		
		if successCount == len(opp.Outcomes) {
			// Update PnL only if all orders were successful
			m.stats.totalPnL += opp.Edge
			log.Printf("Market %s cleared - all orders successful", market.Name)
		} else {
			// Market is cleared for retry since some orders failed
			log.Printf("Market %s cleared - %d/%d orders failed, can retry", market.Name, len(opp.Outcomes)-successCount, len(opp.Outcomes))
		}
	}()
}

// OrderResult represents the result of placing an order
type OrderResult struct {
	TokenID string
	OrderID string
	Success bool
	Error   error
}

func (m Model) viewOrderbook() string {
	// We're viewing a single market's orderbook
	if m.orderbookEvent == nil || m.orderbookTokenID == "" {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			headerStyle.Render("📊 Orderbook View"),
			"",
			dimStyle.Render("Error: Market data not available"),
			"",
			dimStyle.Render("Press 'q' or 'esc' to return"),
		)
	}

	// Get the market name for the selected token
	marketName := "Unknown Market"
	if name, exists := m.orderbookEvent.OutcomeNames[m.orderbookTokenID]; exists {
		marketName = name
	}

	// Header shows the event and specific market (outcome)
	header := headerStyle.Width(m.width).Render(fmt.Sprintf("📊 %s - %s (NO token)", m.orderbookEvent.Name, marketName))

	// Get market data for this specific token
	var marketData *MarketData
	if data, exists := m.marketData.Load(m.orderbookTokenID); exists {
		marketData = data.(*MarketData)
	}

	var content strings.Builder

	// Check if we have full orderbook data from WebSocket
	if m.orderbookData != nil {
		content.WriteString(titleStyle.Render("Full Orderbook") + "\n\n")

		// Sort and display asks (sell orders) - lowest first
		asks := make([]types.OrderSummary, len(m.orderbookData.Asks))
		copy(asks, m.orderbookData.Asks)
		sort.Slice(asks, func(i, j int) bool {
			pi, _ := strconv.ParseFloat(asks[i].Price, 64)
			pj, _ := strconv.ParseFloat(asks[j].Price, 64)
			return pi < pj
		})

		// Display asks in reverse order (highest to lowest) for visual clarity
		content.WriteString(sellStyle.Render("📈 ASKS (Sell Orders)") + "\n")
		content.WriteString(dimStyle.Render("Price         Size") + "\n")
		displayCount := 10
		if len(asks) < displayCount {
			displayCount = len(asks)
		}

		// Show asks from highest to lowest
		for i := displayCount - 1; i >= 0; i-- {
			price, _ := strconv.ParseFloat(asks[i].Price, 64)
			size, _ := strconv.ParseFloat(asks[i].Size, 64)
			content.WriteString(fmt.Sprintf("$%-10.4f %.1f\n", price, size))
		}

		// Calculate and show spread
		if len(asks) > 0 && len(m.orderbookData.Bids) > 0 {
			bestAsk, _ := strconv.ParseFloat(asks[0].Price, 64)

			// Find best bid
			bestBidPrice := 0.0
			for _, bid := range m.orderbookData.Bids {
				price, _ := strconv.ParseFloat(bid.Price, 64)
				if price > bestBidPrice {
					bestBidPrice = price
				}
			}

			spread := bestAsk - bestBidPrice
			spreadPercent := (spread / bestBidPrice) * 100
			content.WriteString(spreadStyle.Render(fmt.Sprintf("\n--- SPREAD: $%.4f (%.2f%%) ---\n\n", spread, spreadPercent)))
		} else {
			content.WriteString("\n\n")
		}

		// Sort and display bids (buy orders) - highest first
		bids := make([]types.OrderSummary, len(m.orderbookData.Bids))
		copy(bids, m.orderbookData.Bids)
		sort.Slice(bids, func(i, j int) bool {
			pi, _ := strconv.ParseFloat(bids[i].Price, 64)
			pj, _ := strconv.ParseFloat(bids[j].Price, 64)
			return pi > pj
		})

		content.WriteString(buyStyle.Render("📉 BIDS (Buy Orders)") + "\n")
		content.WriteString(dimStyle.Render("Price         Size") + "\n")

		displayCount = 10
		if len(bids) < displayCount {
			displayCount = len(bids)
		}

		for i := 0; i < displayCount; i++ {
			price, _ := strconv.ParseFloat(bids[i].Price, 64)
			size, _ := strconv.ParseFloat(bids[i].Size, 64)
			content.WriteString(fmt.Sprintf("$%-10.4f %.1f\n", price, size))
		}

		content.WriteString("\n")
		content.WriteString(dimStyle.Render(fmt.Sprintf("Showing top %d orders | Updated: %s", displayCount, m.orderbookData.Timestamp)) + "\n")
	} else if marketData != nil && !marketData.LastUpdate.IsZero() {
		// Fallback to polling data if no WebSocket data yet
		content.WriteString(titleStyle.Render("Market Summary (Polling Data)") + "\n")
		content.WriteString(dimStyle.Render("Waiting for full orderbook data...") + "\n\n")

		summaryText := fmt.Sprintf("Best Bid: $%.4f (%.1f shares) | Best Ask: $%.4f (%.1f shares)",
			marketData.BestBid, marketData.BestBidSize,
			marketData.BestAsk, marketData.BestAskSize)
		content.WriteString(dimStyle.Render(summaryText) + "\n")

		if marketData.BestBid > 0 && marketData.BestAsk > 0 {
			spread := marketData.BestAsk - marketData.BestBid
			spreadPercent := (spread / marketData.BestBid) * 100
			spreadText := fmt.Sprintf("Spread: $%.4f (%.2f%%)", spread, spreadPercent)
			content.WriteString(spreadStyle.Render(spreadText) + "\n")
		}

		content.WriteString("\n")
		content.WriteString(dimStyle.Render(fmt.Sprintf("Last Update: %s ago", time.Since(marketData.LastUpdate).Round(time.Second).String())) + "\n")
	} else {
		content.WriteString(dimStyle.Render("No data available\n"))
	}

	// Polling status
	pollStatus := "Never polled"
	if !m.lastPollTime.IsZero() {
		timeSincePoll := time.Since(m.lastPollTime)
		statusIcon := "✓"
		if m.lastPollStatus != 200 && m.lastPollStatus != 0 {
			statusIcon = "✗"
		}
		pollStatus = fmt.Sprintf("Last poll: %s ago %s (HTTP %d)",
			timeSincePoll.Round(time.Millisecond),
			statusIcon,
			m.lastPollStatus)
	}

	// Calculate arbitrage for this specific market
	totalCost := 0.0
	liquidCount := 0
	for _, tokenID := range m.orderbookEvent.Outcomes {
		if data, exists := m.marketData.Load(tokenID); exists {
			md := data.(*MarketData)
			if md.BestAsk > 0 {
				totalCost += md.BestAsk
				liquidCount++
			}
		}
	}

	// Arbitrage info
	var arbInfo string
	if liquidCount >= 2 {
		k := float64(liquidCount)
		edge := (k - 1) - totalCost
		edgePercent := (edge / (k - 1)) * 100

		arbInfo = fmt.Sprintf(
			"Arbitrage: Total Cost: $%.4f | Guaranteed: $%.4f | Edge: $%.4f (%.2f%%)",
			totalCost, k-1, edge, edgePercent,
		)

		if edge > 0.01 { // Opportunity threshold
			arbInfo = opportunityStyle.Render("🚨 " + arbInfo)
		} else {
			arbInfo = dimStyle.Render(arbInfo)
		}
	} else {
		arbInfo = dimStyle.Render(fmt.Sprintf("Waiting for liquidity (%d/%d outcomes)", liquidCount, len(m.orderbookEvent.Outcomes)))
	}

	// Compose final view
	finalContent := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		dimStyle.Render(pollStatus),
		"",
		arbInfo,
		"",
		content.String(),
		"",
		dimStyle.Render("Press 'q' or 'esc' to return to overview | Polling at 5 req/s"),
	)

	return finalContent
}

// Channel for orderbook updates
var orderbookChan = make(chan orderbookUpdateMsg, 100)

// WebSocket handler for orderbook updates
type orderbookHandler struct {
	tokenID string
}

func (h *orderbookHandler) OnConnect()    {}
func (h *orderbookHandler) OnDisconnect() {}
func (h *orderbookHandler) OnError(err error) {
	log.Printf("WebSocket error: %v", err)
}
func (h *orderbookHandler) OnOrderBookUpdate(update *websocket.OrderBookUpdate) {
	// Convert WebSocket update to our OrderBookSummary
	book := &types.OrderBookSummary{
		Market:    update.Market,
		AssetID:   update.AssetID,
		Timestamp: update.Timestamp,
		Hash:      update.Hash,
		Bids:      make([]types.OrderSummary, len(update.Buys)),
		Asks:      make([]types.OrderSummary, len(update.Sells)),
	}

	// Copy bids
	for i, buy := range update.Buys {
		book.Bids[i] = types.OrderSummary{
			Price: buy.Price,
			Size:  buy.Size,
		}
	}

	// Copy asks
	for i, sell := range update.Sells {
		book.Asks[i] = types.OrderSummary{
			Price: sell.Price,
			Size:  sell.Size,
		}
	}

	// Send to channel
	select {
	case orderbookChan <- orderbookUpdateMsg{
		tokenID: h.tokenID,
		book:    book,
	}:
	default:
		// Channel full, skip this update
	}
}
func (h *orderbookHandler) OnPriceChange(update *websocket.PriceChangeUpdate)       {}
func (h *orderbookHandler) OnTickSizeChange(update *websocket.TickSizeChangeUpdate) {}
func (h *orderbookHandler) OnLastTradePrice(update *websocket.LastTradePriceUpdate) {}
func (h *orderbookHandler) OnUserUpdate(update *websocket.UserUpdate)               {}

// Subscribe to orderbook updates for a specific token
func (m *Model) subscribeToOrderbook() tea.Cmd {
	return tea.Batch(
		// Start WebSocket subscription
		func() tea.Msg {
			// Create handler
			handler := &orderbookHandler{
				tokenID: m.orderbookTokenID,
			}

			// Subscribe to WebSocket
			wsClient, err := m.client.SubscribeToMarketData([]string{m.orderbookTokenID}, handler)
			if err != nil {
				return logMsg(fmt.Sprintf("Failed to subscribe to orderbook: %v", err))
			}

			m.wsClient = wsClient
			return logMsg("Connected to orderbook WebSocket")
		},
		// Start listening for orderbook updates
		waitForOrderbookUpdate(),
	)
}

// Wait for orderbook updates from the channel
func waitForOrderbookUpdate() tea.Cmd {
	return func() tea.Msg {
		msg := <-orderbookChan
		return msg
	}
}
