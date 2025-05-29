package tests

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// MockMarketData simulates market data for performance testing
type MockMarketData struct {
	tokenID     string
	bestAsk     float64
	bestAskSize float64
	bestBid     float64
	bestBidSize float64
	mutex       sync.RWMutex
}

// MockMarketCache simulates the market cache
type MockMarketCache struct {
	data  map[string]*MockMarketData
	mutex sync.RWMutex
}

func NewMockMarketCache() *MockMarketCache {
	return &MockMarketCache{
		data: make(map[string]*MockMarketData),
	}
}

func (mc *MockMarketCache) Update(tokenID string, data *MockMarketData) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.data[tokenID] = data
}

func (mc *MockMarketCache) Get(tokenID string) (*MockMarketData, bool) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	data, exists := mc.data[tokenID]
	return data, exists
}

// TestArbitrageCaptureLatency tests the latency from opportunity detection to order placement
func TestArbitrageCaptureLatency(t *testing.T) {
	cache := NewMockMarketCache()
	
	// Initialize test market with 3 outcomes
	outcomes := []string{"token1", "token2", "token3"}
	for i, outcome := range outcomes {
		cache.Update(outcome, &MockMarketData{
			tokenID:     outcome,
			bestAsk:     0.30 + float64(i)*0.01, // Prices that sum to < 0.99 (arbitrage opportunity)
			bestAskSize: 100,
			bestBid:     0.29 + float64(i)*0.01,
			bestBidSize: 100,
		})
	}

	// Measure capture to order latency
	latencies := make([]time.Duration, 0, 1000)
	var latencyMutex sync.Mutex

	// Simulate taker arbitrage detection
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			captureTime := time.Now()
			
			// Simulate opportunity detection
			totalCost := 0.0
			for _, outcome := range outcomes {
				data, _ := cache.Get(outcome)
				totalCost += data.bestAsk
			}

			if totalCost < 1.99 { // n-1 where n=3
				// Simulate order placement
				orderTime := time.Now()
				latency := orderTime.Sub(captureTime)
				
				latencyMutex.Lock()
				latencies = append(latencies, latency)
				latencyMutex.Unlock()
			}
		}()
	}
	wg.Wait()

	// Calculate statistics
	if len(latencies) == 0 {
		t.Fatal("No arbitrage opportunities detected")
	}

	total := time.Duration(0)
	maxLatency := time.Duration(0)
	minLatency := latencies[0]

	for _, lat := range latencies {
		total += lat
		if lat > maxLatency {
			maxLatency = lat
		}
		if lat < minLatency {
			minLatency = lat
		}
	}

	avgLatency := total / time.Duration(len(latencies))
	
	t.Logf("Capture to Order Latency Statistics:")
	t.Logf("  Samples: %d", len(latencies))
	t.Logf("  Average: %v", avgLatency)
	t.Logf("  Min: %v", minLatency)
	t.Logf("  Max: %v", maxLatency)

	// Assert that average latency is under 1ms
	if avgLatency > 1*time.Millisecond {
		t.Errorf("Average capture latency too high: %v (expected < 1ms)", avgLatency)
	}
}

// TestConcurrentMarketDataUpdates tests the performance of concurrent market data updates
func TestConcurrentMarketDataUpdates(t *testing.T) {
	cache := NewMockMarketCache()
	
	// Initialize 100 token IDs
	tokenIDs := make([]string, 100)
	for i := 0; i < 100; i++ {
		tokenIDs[i] = fmt.Sprintf("token%d", i)
	}

	// Measure update performance
	start := time.Now()
	
	var wg sync.WaitGroup
	updateCount := int64(0)
	
	// Run for 5 seconds
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(5 * time.Second)
		cancel()
	}()

	// Simulate 10 concurrent updaters
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Update random token
					tokenID := tokenIDs[workerID%len(tokenIDs)]
					cache.Update(tokenID, &MockMarketData{
						tokenID:     tokenID,
						bestAsk:     0.50 + float64(workerID)*0.01,
						bestAskSize: 100,
						bestBid:     0.49 + float64(workerID)*0.01,
						bestBidSize: 100,
					})
					atomic.AddInt64(&updateCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	
	updatesPerSecond := float64(updateCount) / duration.Seconds()
	
	t.Logf("Market Data Update Performance:")
	t.Logf("  Total updates: %d", updateCount)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Updates per second: %.2f", updatesPerSecond)
	
	// Assert minimum performance threshold
	if updatesPerSecond < 10000 {
		t.Errorf("Update rate too low: %.2f updates/sec (expected > 10,000)", updatesPerSecond)
	}
}

// TestOrderExecutionLatency simulates order execution latency
func TestOrderExecutionLatency(t *testing.T) {
	latencies := make([]time.Duration, 0, 1000)
	var mutex sync.Mutex

	// Simulate 1000 order executions
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			start := time.Now()
			
			// Simulate order creation and signing
			time.Sleep(100 * time.Microsecond) // Simulate signing overhead
			
			// Simulate network latency
			time.Sleep(5 * time.Millisecond) // Simulate API call
			
			executionTime := time.Since(start)
			
			mutex.Lock()
			latencies = append(latencies, executionTime)
			mutex.Unlock()
		}()
	}
	wg.Wait()

	// Calculate statistics
	total := time.Duration(0)
	for _, lat := range latencies {
		total += lat
	}
	avgLatency := total / time.Duration(len(latencies))
	
	// Calculate P99
	p99Index := int(float64(len(latencies)) * 0.99)
	p99Latency := latencies[p99Index]
	
	t.Logf("Order Execution Latency:")
	t.Logf("  Average: %v", avgLatency)
	t.Logf("  P99: %v", p99Latency)
	
	// Assert performance thresholds
	if avgLatency > 10*time.Millisecond {
		t.Errorf("Average execution latency too high: %v (expected < 10ms)", avgLatency)
	}
	if p99Latency > 20*time.Millisecond {
		t.Errorf("P99 execution latency too high: %v (expected < 20ms)", p99Latency)
	}
}

// BenchmarkArbitrageDetection benchmarks the arbitrage detection algorithm
func BenchmarkArbitrageDetection(b *testing.B) {
	cache := NewMockMarketCache()
	
	// Initialize test market
	outcomes := []string{"token1", "token2", "token3"}
	for i, outcome := range outcomes {
		cache.Update(outcome, &MockMarketData{
			tokenID:     outcome,
			bestAsk:     0.30 + float64(i)*0.01,
			bestAskSize: 100,
		})
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Simulate arbitrage detection
		totalCost := 0.0
		for _, outcome := range outcomes {
			data, _ := cache.Get(outcome)
			totalCost += data.bestAsk
		}
		
		if totalCost < 1.99 {
			// Arbitrage opportunity detected
			_ = totalCost
		}
	}
}

// BenchmarkConcurrentCacheAccess benchmarks concurrent cache access
func BenchmarkConcurrentCacheAccess(b *testing.B) {
	cache := NewMockMarketCache()
	
	// Initialize 100 tokens
	for i := 0; i < 100; i++ {
		tokenID := fmt.Sprintf("token%d", i)
		cache.Update(tokenID, &MockMarketData{
			tokenID:     tokenID,
			bestAsk:     0.50,
			bestAskSize: 100,
		})
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			tokenID := fmt.Sprintf("token%d", i%100)
			_, _ = cache.Get(tokenID)
			i++
		}
	})
}

// TestMemoryEfficiency tests memory usage under load
func TestMemoryEfficiency(t *testing.T) {
	cache := NewMockMarketCache()
	
	// Add 10,000 tokens
	for i := 0; i < 10000; i++ {
		tokenID := fmt.Sprintf("token%d", i)
		cache.Update(tokenID, &MockMarketData{
			tokenID:     tokenID,
			bestAsk:     0.50,
			bestAskSize: 100,
			bestBid:     0.49,
			bestBidSize: 100,
		})
	}

	// Verify all tokens are accessible
	for i := 0; i < 10000; i++ {
		tokenID := fmt.Sprintf("token%d", i)
		if _, exists := cache.Get(tokenID); !exists {
			t.Errorf("Token %s not found in cache", tokenID)
		}
	}

	t.Logf("Successfully stored and accessed 10,000 tokens")
}