package main

import (
	"fmt"
	"log"
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
type orderBookMsg *websocket.OrderBookUpdate
type priceChangeMsg *websocket.PriceChangeUpdate
type tickSizeChangeMsg *websocket.TickSizeChangeUpdate
type userUpdateMsg *websocket.UserUpdate
type tickMsg time.Time
type errorMsg error

// Model represents the application state
type model struct {
	help         help.Model
	keys         keyMap
	currentBook  *websocket.OrderBookUpdate
	lastUpdate   time.Time
	startTime    time.Time
	heartbeat    int
	priceChanges map[string]time.Time
	width        int
	height       int
	wsClient     *websocket.Client
	connected    bool
	lastError    error
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
	h.program.Send(orderBookMsg(update))
}

func (h *wsHandler) OnPriceChange(update *websocket.PriceChangeUpdate) {
	h.program.Send(priceChangeMsg(update))
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
		m.currentBook = (*websocket.OrderBookUpdate)(msg)
		m.lastUpdate = time.Now()
		m.connected = true
		m.updateCounts.orderBook++
		m.updateCounts.total++

	case priceChangeMsg:
		m.updateCounts.priceChange++
		m.updateCounts.total++
		if m.currentBook != nil {
			update := (*websocket.PriceChangeUpdate)(msg)
			// Apply price changes
			for _, change := range update.Changes {
				m.applyPriceChange(change)
				m.priceChanges[change.Price] = time.Now()
			}
			m.currentBook.Hash = update.Hash
			m.currentBook.Timestamp = update.Timestamp
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
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Header
	title := titleStyle.Render("Meeep's CLOB client - Realtime Book View")
	b.WriteString(title + "\n\n")

	if m.currentBook == nil {
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

	// Market info
	var timestamp time.Time
	if m.currentBook.Timestamp != "" {
		if millis, err := strconv.ParseInt(m.currentBook.Timestamp, 10, 64); err == nil {
			timestamp = time.Unix(millis/1000, (millis%1000)*1000000)
		} else {
			timestamp = time.Now()
		}
	} else {
		timestamp = time.Now()
	}

	marketInfo := fmt.Sprintf("Market: %s | Asset: %s | Time: %s",
		m.currentBook.Market, m.currentBook.AssetID, timestamp.Format("15:04:05"))
	b.WriteString(headerStyle.Render(marketInfo) + "\n")

	// Update statistics
	updateInfo := fmt.Sprintf("Updates Received: %d total (📊 %d books, 🔄 %d changes, ⚠️  %d errors)",
		m.updateCounts.total, m.updateCounts.orderBook, m.updateCounts.priceChange, m.updateCounts.error)
	if m.updateCounts.tickSize > 0 || m.updateCounts.user > 0 {
		updateInfo += fmt.Sprintf(" | 📏 %d tick, 👤 %d user", m.updateCounts.tickSize, m.updateCounts.user)
	}
	b.WriteString(statusStyle.Render(updateInfo) + "\n\n")

	// Sells (Asks) - show highest prices first, but display in reverse
	sellsTitle := sellStyle.Render("📈 SELLS (Asks)")
	b.WriteString(sellsTitle + "\n")
	b.WriteString(m.renderOrderSide(m.currentBook.Sells, false, true) + "\n")

	// Spread
	spreadText := m.calculateSpread()
	b.WriteString(spreadStyle.Render("💰 "+spreadText) + "\n\n")

	// Buys (Bids) - show highest prices first
	buysTitle := buyStyle.Render("📉 BUYS (Bids)")
	b.WriteString(buysTitle + "\n")
	b.WriteString(m.renderOrderSide(m.currentBook.Buys, true, false) + "\n")

	// Status bar
	indicator := m.getHeartbeatIndicator()
	uptime := time.Since(m.startTime).Truncate(time.Second)
	status := fmt.Sprintf("%s LIVE | %d buys, %d sells | Last: %s | Uptime: %s | 'q' to quit",
		indicator, len(m.currentBook.Buys), len(m.currentBook.Sells),
		m.lastUpdate.Format("15:04:05"), uptime)

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

func (m model) calculateSpread() string {
	if len(m.currentBook.Buys) > 0 && len(m.currentBook.Sells) > 0 {
		bestBuy, _ := strconv.ParseFloat(m.currentBook.Buys[0].Price, 64)
		bestSell, _ := strconv.ParseFloat(m.currentBook.Sells[0].Price, 64)
		spread := bestSell - bestBuy
		spreadPercent := (spread / bestBuy) * 100
		return fmt.Sprintf("SPREAD: %.4f (%.2f%%)", spread, spreadPercent)
	}
	return "SPREAD: No data"
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

	// Header with proper spacing - using exact character positions
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Bold(true)
	header := headerStyle.Render("  PRICE        SIZE       TOTAL      DEPTH")
	b.WriteString(header + "\n")

	// Display up to 10 orders
	maxRows := 10
	if len(sortedOrders) < maxRows {
		maxRows = len(sortedOrders)
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

		// Create depth bar
		depthPercent := (size / runningTotal) * 100
		barLength := int(depthPercent / 10)
		if barLength > 10 {
			barLength = 10
		}
		depthBar := strings.Repeat("█", barLength) + strings.Repeat("░", 10-barLength)

		// Render each column with proper alignment
		styledPrice := priceStyle.Render(priceText)
		styledSize := sizeStyle.Render(sizeText)
		styledTotal := totalStyle.Render(totalText)

		// Combine columns with proper spacing
		line := "  " + styledPrice + " " + styledSize + " " + styledTotal + " " + depthBar
		b.WriteString(line + "\n")
	}

	return b.String()
}

func (m *model) applyPriceChange(change websocket.PriceChange) {
	if m.currentBook == nil {
		return
	}

	// Apply change to the appropriate side
	if change.Side == "BUY" {
		m.updateOrderSide(&m.currentBook.Buys, change.Price, change.Size)
	} else if change.Side == "SELL" {
		m.updateOrderSide(&m.currentBook.Sells, change.Price, change.Size)
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
	// Create the tea program
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())

	// Create WebSocket handler that can send messages to the program
	handler := &wsHandler{program: p}

	// Start WebSocket connection in a goroutine
	go func() {
		time.Sleep(500 * time.Millisecond) // Give the UI time to start

		// Create CLOB client
		clobClient, err := client.NewClobClient("https://clob.polymarket.com", 137, "", nil, nil, nil)
		if err != nil {
			handler.program.Send(errorMsg(err))
			return
		}

		// Token ID for a popular market
		tokenID := "64857617821618792090309776061594999588607561964140319397152984325528949636614"

		// Subscribe to real-time market data
		_, err = clobClient.SubscribeToMarketData([]string{tokenID}, handler)
		if err != nil {
			handler.program.Send(errorMsg(err))
			return
		}

		// Keep the connection alive
		for {
			time.Sleep(1 * time.Second)
		}
	}()

	// Run the program
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
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
