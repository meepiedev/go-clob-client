package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pooofdevelopment/go-clob-client/pkg/client"
	"github.com/pooofdevelopment/go-clob-client/pkg/types"
	"github.com/pooofdevelopment/go-clob-client/pkg/websocket"
)

// Colors and styles
var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Padding(0, 1)
	sellStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	buyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	spreadStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true).Padding(0, 1)
	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true)
	borderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
	pulseStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
)

// Key bindings
type keyMap struct {
	quit key.Binding
}

var keys = keyMap{
	quit: key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// Messages for the tea program
type orderBookMsg struct {
	update  *websocket.OrderBookUpdate
	tokenID string
}
type priceChangeMsg struct {
	update  *websocket.PriceChangeUpdate
	tokenID string
}
type tickSizeChangeMsg *websocket.TickSizeChangeUpdate
type userUpdateMsg *websocket.UserUpdate
type marketInfoMsg *types.GammaMarket
type yesSpreadMsg string
type noSpreadMsg string
type tickMsg time.Time
type errorMsg error

// Model represents the application state
type model struct {
	help         help.Model
	keys         keyMap
	yesBook      *websocket.OrderBookUpdate
	noBook       *websocket.OrderBookUpdate
	lastUpdate   time.Time
	startTime    time.Time
	heartbeat    int
	priceChanges map[string]time.Time
	width        int
	height       int
	wsClient     *websocket.Client
	connected    bool
	lastError    error
	clobClient   *client.ClobClient
	marketSlug   string
	yesTokenID   string
	noTokenID    string
	yesSpread    string
	noSpread     string
	marketInfo   *types.GammaMarket
	// Update counters
	updateCounts struct {
		orderBook   int
		priceChange int
		tickSize    int
		user        int
		error       int
		total       int
	}
}

// WebSocket handler that sends messages to the tea program
type wsHandler struct {
	program *tea.Program
}

func (h *wsHandler) OnConnect() {
	// Connection handled in the program loop
}

func (h *wsHandler) OnDisconnect() {
	// Disconnection handled in the program loop
}

func (h *wsHandler) OnError(err error) {
	h.program.Send(errorMsg(err))
}

func (h *wsHandler) OnOrderBookUpdate(update *websocket.OrderBookUpdate) {
	h.program.Send(orderBookMsg{update: update, tokenID: update.AssetID})
}

func (h *wsHandler) OnPriceChange(update *websocket.PriceChangeUpdate) {
	h.program.Send(priceChangeMsg{update: update, tokenID: update.AssetID})
}

func (h *wsHandler) OnTickSizeChange(update *websocket.TickSizeChangeUpdate) {
	h.program.Send(tickSizeChangeMsg(update))
}

func (h *wsHandler) OnUserUpdate(update *websocket.UserUpdate) {
	h.program.Send(userUpdateMsg(update))
}

func initialModel() model {
	return model{
		help:         help.New(),
		keys:         keys,
		startTime:    time.Now(),
		priceChanges: make(map[string]time.Time),
		connected:    false,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		connectCmd(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, m.keys.quit) {
			if m.wsClient != nil {
				m.wsClient.Close()
			}
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		m.heartbeat++
		return m, tickCmd()

	case orderBookMsg:
		// Route to correct book based on token ID
		if msg.tokenID == m.yesTokenID {
			m.yesBook = msg.update
		} else if msg.tokenID == m.noTokenID {
			m.noBook = msg.update
		}
		m.lastUpdate = time.Now()
		m.connected = true
		m.updateCounts.orderBook++
		m.updateCounts.total++

	case priceChangeMsg:
		m.updateCounts.priceChange++
		m.updateCounts.total++
		// Route to correct book based on token ID
		var targetBook *websocket.OrderBookUpdate
		if msg.tokenID == m.yesTokenID {
			targetBook = m.yesBook
		} else if msg.tokenID == m.noTokenID {
			targetBook = m.noBook
		}

		if targetBook != nil {
			// Apply price changes
			for _, change := range msg.update.Changes {
				m.applyPriceChangeToBook(targetBook, change)
				m.priceChanges[change.Price] = time.Now()
			}
			targetBook.Hash = msg.update.Hash
			targetBook.Timestamp = msg.update.Timestamp
			m.lastUpdate = time.Now()
		}

	case errorMsg:
		m.lastError = error(msg)
		m.connected = false
		m.updateCounts.error++
		m.updateCounts.total++

	case tickSizeChangeMsg:
		m.updateCounts.tickSize++
		m.updateCounts.total++

	case userUpdateMsg:
		m.updateCounts.user++
		m.updateCounts.total++

	case marketInfoMsg:
		m.marketInfo = (*types.GammaMarket)(msg)

	case yesSpreadMsg:
		m.yesSpread = string(msg)

	case noSpreadMsg:
		m.noSpread = string(msg)
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Header
	title := titleStyle.Render("Meeep's CLOB client - Dual Market View")
	b.WriteString(title + "\n\n")

	if m.yesBook == nil && m.noBook == nil {
		indicator := m.getHeartbeatIndicator()
		uptime := time.Since(m.startTime).Truncate(time.Second)
		status := fmt.Sprintf("%s Connecting... | Uptime: %s", indicator, uptime)
		b.WriteString(statusStyle.Render(status))

		// Show update counts even while connecting
		if m.updateCounts.total > 0 {
			updateInfo := fmt.Sprintf("\nUpdates Received: %d total (📊 %d books, 🔄 %d changes, ⚠️ %d errors)",
				m.updateCounts.total, m.updateCounts.orderBook, m.updateCounts.priceChange, m.updateCounts.error)
			b.WriteString(statusStyle.Render(updateInfo))
		}

		return b.String()
	}

	// Market info - use market info if available, otherwise fall back to book data
	var marketInfo string
	if m.marketInfo != nil {
		marketInfo = fmt.Sprintf("Market: %s | Slug: %s | Last Update: %s",
			m.marketInfo.Question, m.marketInfo.Slug, m.lastUpdate.Format("15:04:05"))
	} else {
		// Fallback to book data
		var book *websocket.OrderBookUpdate
		if m.yesBook != nil {
			book = m.yesBook
		} else if m.noBook != nil {
			book = m.noBook
		}

		if book != nil {
			var timestamp time.Time
			if book.Timestamp != "" {
				if millis, err := strconv.ParseInt(book.Timestamp, 10, 64); err == nil {
					timestamp = time.Unix(millis/1000, (millis%1000)*1000000)
				} else {
					timestamp = time.Now()
				}
			} else {
				timestamp = time.Now()
			}

			marketInfo = fmt.Sprintf("Market: %s | Time: %s", book.Market, timestamp.Format("15:04:05"))
		} else {
			marketInfo = fmt.Sprintf("Market: %s | Time: %s", m.marketSlug, time.Now().Format("15:04:05"))
		}
	}
	b.WriteString(headerStyle.Render(marketInfo) + "\n")

	// Update statistics
	updateInfo := fmt.Sprintf("Updates Received: %d total (📊 %d books, 🔄 %d changes, ⚠️  %d errors)",
		m.updateCounts.total, m.updateCounts.orderBook, m.updateCounts.priceChange, m.updateCounts.error)
	if m.updateCounts.tickSize > 0 || m.updateCounts.user > 0 {
		updateInfo += fmt.Sprintf(" | 📏 %d tick, 👤 %d user", m.updateCounts.tickSize, m.updateCounts.user)
	}
	b.WriteString(statusStyle.Render(updateInfo) + "\n\n")

	// Render side-by-side books
	leftCol := m.renderBookColumn("YES", m.yesBook, m.yesSpread)
	rightCol := m.renderBookColumn("NO", m.noBook, m.noSpread)

	// Split into lines and combine side by side
	leftLines := strings.Split(leftCol, "\n")
	rightLines := strings.Split(rightCol, "\n")

	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}

	// Pad shorter column with empty lines
	for len(leftLines) < maxLines {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxLines {
		rightLines = append(rightLines, "")
	}

	// Combine columns side by side with proper width handling
	for i := 0; i < maxLines; i++ {
		leftLine := leftLines[i]
		rightLine := rightLines[i]

		// Calculate visual width (excluding ANSI codes) and pad to 70 characters
		leftVisualWidth := lipgloss.Width(leftLine)
		padding := 70 - leftVisualWidth
		if padding < 0 {
			padding = 0
		}

		paddedLeft := leftLine + strings.Repeat(" ", padding)
		b.WriteString(paddedLeft + " |   " + rightLine + "\n")
	}

	// Status bar
	indicator := m.getHeartbeatIndicator()
	uptime := time.Since(m.startTime).Truncate(time.Second)

	yesBuys, yesSells := 0, 0
	noBuys, noSells := 0, 0

	if m.yesBook != nil {
		yesBuys = len(m.yesBook.Buys)
		yesSells = len(m.yesBook.Sells)
	}
	if m.noBook != nil {
		noBuys = len(m.noBook.Buys)
		noSells = len(m.noBook.Sells)
	}

	status := fmt.Sprintf("%s LIVE | YES: %d/%d | NO: %d/%d | Last: %s | Uptime: %s | 'q' to quit | Debug: tail -f orderbook_debug.log",
		indicator, yesBuys, yesSells, noBuys, noSells, m.lastUpdate.Format("15:04:05"), uptime)

	if m.lastError != nil {
		status = fmt.Sprintf("❌ ERROR: %v | 'q' to quit", m.lastError)
	}

	b.WriteString("\n" + statusStyle.Render(status))

	return b.String()
}

func (m model) getHeartbeatIndicator() string {
	indicators := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return pulseStyle.Render(indicators[m.heartbeat%len(indicators)])
}

func (m model) renderBookColumn(title string, book *websocket.OrderBookUpdate, spread string) string {
	var b strings.Builder

	// Column title
	var titleColor lipgloss.Style
	if title == "YES" {
		titleColor = buyStyle
	} else {
		titleColor = sellStyle
	}
	b.WriteString(titleColor.Render(fmt.Sprintf("📊 %s TOKEN", title)) + "\n")

	if book == nil {
		b.WriteString("  No data available\n")
		return b.String()
	}

	// Asks (highest prices first, displayed in reverse)
	b.WriteString(sellStyle.Render("📈 ASKS") + "\n")
	b.WriteString(m.renderOrderSide(book.Sells, false, true) + "\n")

	// Spread calculation
	spreadText := "SPREAD: No data"
	if spread != "" {
		spreadText = fmt.Sprintf("SPREAD (API): %s", spread)
	} else if len(book.Buys) > 0 && len(book.Sells) > 0 {
		// Sort to ensure we get the best prices
		sortedBuys := make([]types.OrderSummary, len(book.Buys))
		copy(sortedBuys, book.Buys)
		sort.Slice(sortedBuys, func(i, j int) bool {
			priceI, _ := strconv.ParseFloat(sortedBuys[i].Price, 64)
			priceJ, _ := strconv.ParseFloat(sortedBuys[j].Price, 64)
			return priceI > priceJ // Highest bid first
		})

		sortedSells := make([]types.OrderSummary, len(book.Sells))
		copy(sortedSells, book.Sells)
		sort.Slice(sortedSells, func(i, j int) bool {
			priceI, _ := strconv.ParseFloat(sortedSells[i].Price, 64)
			priceJ, _ := strconv.ParseFloat(sortedSells[j].Price, 64)
			return priceI < priceJ // Lowest ask first
		})

		bestBid, _ := strconv.ParseFloat(sortedBuys[0].Price, 64)
		bestAsk, _ := strconv.ParseFloat(sortedSells[0].Price, 64)
		spreadVal := bestAsk - bestBid
		spreadPercent := (spreadVal / bestBid) * 100
		spreadText = fmt.Sprintf("SPREAD (Local): %.4f (%.2f%%)", spreadVal, spreadPercent)
	}
	b.WriteString(spreadStyle.Render("💰 "+spreadText) + "\n\n")

	// Bids (highest prices first)
	b.WriteString(buyStyle.Render("📉 BIDS") + "\n")
	b.WriteString(m.renderOrderSide(book.Buys, true, false) + "\n")

	return b.String()
}

func (m model) renderOrderSide(orders []types.OrderSummary, isBuys bool, reverse bool) string {
	if len(orders) == 0 {
		return "  No orders"
	}

	// Sort orders by price
	sortedOrders := make([]types.OrderSummary, len(orders))
	copy(sortedOrders, orders)

	if isBuys {
		// Buys: highest price first
		sort.Slice(sortedOrders, func(i, j int) bool {
			priceI, _ := strconv.ParseFloat(sortedOrders[i].Price, 64)
			priceJ, _ := strconv.ParseFloat(sortedOrders[j].Price, 64)
			return priceI > priceJ
		})
	} else {
		// Sells: lowest price first
		sort.Slice(sortedOrders, func(i, j int) bool {
			priceI, _ := strconv.ParseFloat(sortedOrders[i].Price, 64)
			priceJ, _ := strconv.ParseFloat(sortedOrders[j].Price, 64)
			return priceI < priceJ
		})
	}

	// If reverse is true, reverse the order (for sells display)
	if reverse {
		for i := len(sortedOrders)/2 - 1; i >= 0; i-- {
			opp := len(sortedOrders) - 1 - i
			sortedOrders[i], sortedOrders[opp] = sortedOrders[opp], sortedOrders[i]
		}
	}

	var b strings.Builder

	// Header with proper spacing to match data columns
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Bold(true)
	headerLine := fmt.Sprintf("  %s %s %s %s %s",
		lipgloss.NewStyle().Width(12).Render("PRICE"),
		lipgloss.NewStyle().Width(10).Render("SIZE"),
		lipgloss.NewStyle().Width(10).Render("TOTAL"),
		lipgloss.NewStyle().Width(10).Render("VOL. BAR"),
		"DEPTH")
	header := headerStyle.Render(headerLine)
	b.WriteString(header + "\n")

	// Display up to 10 orders
	maxRows := 10
	if len(sortedOrders) < maxRows {
		maxRows = len(sortedOrders)
	}

	// First pass: find the maximum individual order size and total volume
	var maxSize, totalVolume float64
	ordersToShow := sortedOrders
	if len(ordersToShow) > maxRows {
		ordersToShow = ordersToShow[:maxRows]
	}

	for _, order := range ordersToShow {
		if size, err := strconv.ParseFloat(order.Size, 64); err == nil {
			if size > maxSize {
				maxSize = size
			}
			totalVolume += size
		}
	}

	var runningTotal float64
	for i := 0; i < maxRows; i++ {
		order := sortedOrders[i]
		price, _ := strconv.ParseFloat(order.Price, 64)
		size, _ := strconv.ParseFloat(order.Size, 64)
		runningTotal += size

		// Format values with fixed widths
		priceText := fmt.Sprintf("$%.4f", price)
		sizeText := formatSize(size)
		totalText := formatSize(runningTotal)

		// Create styled columns with fixed widths using lipgloss
		var priceStyle, sizeStyle, totalStyle lipgloss.Style
		if changeTime, exists := m.priceChanges[order.Price]; exists && time.Since(changeTime) < 3*time.Second {
			// Highlight recent changes
			priceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true).Width(12).Align(lipgloss.Left)
			sizeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true).Width(10).Align(lipgloss.Right)
			totalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Width(10).Align(lipgloss.Right)
		} else {
			if isBuys {
				priceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true).Width(12).Align(lipgloss.Left)
				sizeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true).Width(10).Align(lipgloss.Right)
				totalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Width(10).Align(lipgloss.Right)
			} else {
				priceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Width(12).Align(lipgloss.Left)
				sizeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Width(10).Align(lipgloss.Right)
				totalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Width(10).Align(lipgloss.Right)
			}
		}

		// Create SIZE BAR showing individual order size relative to max size
		var sizeBarLength int
		if maxSize > 0 {
			// Each order gets bar length proportional to its size vs the largest order
			// Minimum bar length of 1 to ensure visibility
			sizePercent := (size / maxSize) * 100
			sizeBarLength = int(sizePercent / 10)
			if sizeBarLength < 1 && size > 0 {
				sizeBarLength = 1 // Ensure even small orders get at least 1 block
			}
			if sizeBarLength > 10 {
				sizeBarLength = 10
			}
		}
		sizeBar := strings.Repeat("█", sizeBarLength) + strings.Repeat("░", 10-sizeBarLength)

		// Create DEPTH bar showing cumulative market depth
		var depthBarLength int
		if totalVolume > 0 {
			// Shows what percentage of total volume this cumulative total represents
			depthPercent := (runningTotal / totalVolume) * 100
			depthBarLength = int(depthPercent / 10)
			if depthBarLength > 10 {
				depthBarLength = 10
			}
		}
		depthBar := strings.Repeat("█", depthBarLength) + strings.Repeat("░", 10-depthBarLength)

		// Render each column with proper alignment
		styledPrice := priceStyle.Render(priceText)
		styledSize := sizeStyle.Render(sizeText)
		styledTotal := totalStyle.Render(totalText)

		// Combine columns with proper spacing - styles already have width set
		line := fmt.Sprintf("  %s %s %s %s %s",
			styledPrice, styledSize, styledTotal, sizeBar, depthBar)
		b.WriteString(line + "\n")
	}

	return b.String()
}

func (m *model) applyPriceChangeToBook(book *websocket.OrderBookUpdate, change websocket.PriceChange) {
	if book == nil {
		return
	}

	// Apply change to the appropriate side
	if change.Side == "BUY" {
		m.updateOrderSide(&book.Buys, change.Price, change.Size)
	} else if change.Side == "SELL" {
		m.updateOrderSide(&book.Sells, change.Price, change.Size)
	}
}

func (m *model) updateOrderSide(orders *[]types.OrderSummary, price, size string) {
	// Find existing price level
	for i, order := range *orders {
		if order.Price == price {
			if size == "0" {
				// Remove the price level
				*orders = append((*orders)[:i], (*orders)[i+1:]...)
			} else {
				// Update the size
				(*orders)[i].Size = size
			}
			return
		}
	}

	// If price level doesn't exist and size > 0, add it
	if size != "0" {
		newOrder := types.OrderSummary{Price: price, Size: size}
		*orders = append(*orders, newOrder)
	}
}

// Commands
func tickCmd() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func connectCmd() tea.Cmd {
	return func() tea.Msg {
		// This will be handled in main() after the program starts
		return nil
	}
}

func main() {
	// Setup logging to file instead of stdout to keep TUI clean
	logFile, err := os.OpenFile("orderbook_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Create CLOB client first
	clobClient, err := client.NewClobClient("https://clob.polymarket.com", 137, "", nil, nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Market slug - you can change this to any market
	marketSlug := "us-recession-in-2025"

	// Look up market by slug and get token IDs
	log.Printf("Looking up market: %s", marketSlug)
	marketInfo, yesTokenID, noTokenID, err := lookupMarketBySlug(clobClient, marketSlug)
	if err != nil {
		log.Fatalf("Failed to lookup market: %v", err)
	}

	log.Printf("Found market: %s", marketInfo.Question)
	log.Printf("YES Token: %s", yesTokenID)
	log.Printf("NO Token: %s", noTokenID)

	// Create the tea program with client and token IDs
	initialModelWithClient := func() model {
		m := initialModel()
		m.clobClient = clobClient
		m.marketSlug = marketSlug
		m.yesTokenID = yesTokenID
		m.noTokenID = noTokenID
		m.marketInfo = marketInfo
		return m
	}

	log.Printf("Starting TUI...")
	p := tea.NewProgram(initialModelWithClient(), tea.WithAltScreen())

	// Create WebSocket handler that can send messages to the program
	handler := &wsHandler{program: p}

	// Start background processes
	log.Printf("Starting background processes...")

	// Start WebSocket connection in a goroutine
	go func() {
		log.Printf("Connecting to WebSocket...")
		time.Sleep(500 * time.Millisecond) // Give the UI time to start

		// Subscribe to real-time market data for both tokens
		_, err = clobClient.SubscribeToMarketData([]string{yesTokenID, noTokenID}, handler)
		if err != nil {
			log.Printf("WebSocket connection failed: %v", err)
			handler.program.Send(errorMsg(err))
			return
		}
		log.Printf("WebSocket connected successfully")

		// Keep the connection alive
		for {
			time.Sleep(1 * time.Second)
		}
	}()

	// Start periodic spread fetching for both tokens
	go func() {
		time.Sleep(2 * time.Second) // Wait a bit for initial connection
		log.Printf("Starting spread fetching...")

		for {
			// Fetch YES spread
			if resp, err := clobClient.GetSpread(yesTokenID); err == nil {
				log.Printf("YES spread API response: %+v", resp)
				if spreadStr := formatSpreadResponse(resp); spreadStr != "" {
					log.Printf("YES spread formatted: %s", spreadStr)
					p.Send(yesSpreadMsg(spreadStr))
				}
			} else {
				log.Printf("Failed to fetch YES spread: %v", err)
			}

			// Fetch NO spread
			if resp, err := clobClient.GetSpread(noTokenID); err == nil {
				log.Printf("NO spread API response: %+v", resp)
				if spreadStr := formatSpreadResponse(resp); spreadStr != "" {
					log.Printf("NO spread formatted: %s", spreadStr)
					p.Send(noSpreadMsg(spreadStr))
				}
			} else {
				log.Printf("Failed to fetch NO spread: %v", err)
			}

			time.Sleep(10 * time.Second) // Update spreads every 10 seconds
		}
	}()

	// Send initial market info to UI
	go func() {
		time.Sleep(100 * time.Millisecond)
		p.Send(marketInfoMsg(marketInfo))
	}()

	// Run the program
	log.Printf("Running TUI program...")
	if _, err := p.Run(); err != nil {
		log.Fatalf("TUI program failed: %v", err)
	}
	log.Printf("TUI program exited")
}

// lookupMarketBySlug finds a market by slug and returns token IDs
func lookupMarketBySlug(client *client.ClobClient, slug string) (*types.GammaMarket, string, string, error) {
	// Search for the market by slug
	params := &types.GammaMarketsParams{
		Slug:  []string{slug},
		Limit: 1,
	}

	markets, err := client.GetGammaMarkets(params)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to search markets: %w", err)
	}

	if len(markets) == 0 {
		return nil, "", "", fmt.Errorf("market with slug '%s' not found", slug)
	}

	market := markets[0]

	if len(market.ClobTokenIDs) < 2 {
		return nil, "", "", fmt.Errorf("market '%s' does not have both YES and NO tokens", slug)
	}

	// Typically the first token is YES and second is NO, but this can vary
	// For safety, we'll just use them in order
	yesTokenID := market.ClobTokenIDs[0]
	noTokenID := market.ClobTokenIDs[1]

	return &market, yesTokenID, noTokenID, nil
}

// formatSpreadResponse formats the API spread response
func formatSpreadResponse(resp map[string]interface{}) string {
	// Try different possible response structures

	// Method 1: Direct spread field
	if spread, ok := resp["spread"].(float64); ok {
		if bid, bidOk := resp["bid"].(float64); bidOk && bid > 0 {
			percentage := (spread / bid) * 100
			return fmt.Sprintf("%.4f (%.2f%%)", spread, percentage)
		}
		if mid, midOk := resp["mid"].(float64); midOk && mid > 0 {
			percentage := (spread / mid) * 100
			return fmt.Sprintf("%.4f (%.2f%%)", spread, percentage)
		}
		return fmt.Sprintf("%.4f", spread)
	}

	// Method 2: String spread field
	if spreadStr, ok := resp["spread"].(string); ok {
		if spread, err := strconv.ParseFloat(spreadStr, 64); err == nil {
			if bid, bidOk := resp["bid"].(float64); bidOk && bid > 0 {
				percentage := (spread / bid) * 100
				return fmt.Sprintf("%.4f (%.2f%%)", spread, percentage)
			}
		}
		return spreadStr
	}

	// Method 3: Calculate from bid/ask
	if bid, bidOk := resp["bid"].(float64); bidOk {
		if ask, askOk := resp["ask"].(float64); askOk {
			spread := ask - bid
			percentage := (spread / bid) * 100
			return fmt.Sprintf("%.4f (%.2f%%)", spread, percentage)
		}
	}

	// Method 4: Try nested data structure
	if data, ok := resp["data"].(map[string]interface{}); ok {
		return formatSpreadResponse(data)
	}

	// Method 5: Check for individual price fields
	if buyPrice, buyOk := resp["buy_price"].(float64); buyOk {
		if sellPrice, sellOk := resp["sell_price"].(float64); sellOk {
			spread := sellPrice - buyPrice
			percentage := (spread / buyPrice) * 100
			return fmt.Sprintf("%.4f (%.2f%%)", spread, percentage)
		}
	}

	// Method 6: Check for best_bid/best_ask
	if bestBid, bidOk := resp["best_bid"].(float64); bidOk {
		if bestAsk, askOk := resp["best_ask"].(float64); askOk {
			spread := bestAsk - bestBid
			percentage := (spread / bestBid) * 100
			return fmt.Sprintf("%.4f (%.2f%%)", spread, percentage)
		}
	}

	return ""
}

// formatSize formats large numbers with appropriate suffixes
func formatSize(size float64) string {
	if size >= 1000000 {
		return fmt.Sprintf("%.1fM", size/1000000)
	} else if size >= 1000 {
		return fmt.Sprintf("%.1fK", size/1000)
	} else if size >= 1 {
		return fmt.Sprintf("%.0f", size)
	} else {
		return fmt.Sprintf("%.3f", size)
	}
}
