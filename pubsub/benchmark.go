package pubsub

import (
	"fmt"
	"sync"
	"time"

	"github.com/tphuc/pubsub/model"
)

// BenchmarkConfig holds configuration for the benchmark
type BenchmarkConfig struct {
	MessageCount        int           // Number of messages to publish
	ConcurrentProducers int           // Number of concurrent producers
	ConcurrentConsumers int           // Number of concurrent consumers
	MessageSize         int           // Size of each message in bytes (approximate)
	WarmupCount         int           // Number of warmup messages before measurement
	RunDuration         time.Duration // Duration to run the benchmark
}

// NewBenchmarkConfig creates a default benchmark configuration
func NewBenchmarkConfig() BenchmarkConfig {
	return BenchmarkConfig{
		MessageCount:        10000,
		ConcurrentProducers: 1,
		ConcurrentConsumers: 1,
		MessageSize:         1024, // 1KB
		WarmupCount:         100,
		RunDuration:         10 * time.Second,
	}
}

// WithMessageCount sets the number of messages to publish
func (c BenchmarkConfig) WithMessageCount(count int) BenchmarkConfig {
	c.MessageCount = count
	return c
}

// WithConcurrentProducers sets the number of concurrent producers
func (c BenchmarkConfig) WithConcurrentProducers(count int) BenchmarkConfig {
	c.ConcurrentProducers = count
	return c
}

// WithConcurrentConsumers sets the number of concurrent consumers
func (c BenchmarkConfig) WithConcurrentConsumers(count int) BenchmarkConfig {
	c.ConcurrentConsumers = count
	return c
}

// WithMessageSize sets the approximate size of each message in bytes
func (c BenchmarkConfig) WithMessageSize(size int) BenchmarkConfig {
	c.MessageSize = size
	return c
}

// WithWarmupCount sets the number of warmup messages
func (c BenchmarkConfig) WithWarmupCount(count int) BenchmarkConfig {
	c.WarmupCount = count
	return c
}

// WithRunDuration sets the duration to run the benchmark
func (c BenchmarkConfig) WithRunDuration(duration time.Duration) BenchmarkConfig {
	c.RunDuration = duration
	return c
}

// BenchmarkResult holds the results of a benchmark run
type BenchmarkResult struct {
	Config            BenchmarkConfig
	TotalMessages     int
	TotalDuration     time.Duration
	MessagesPerSecond float64
	AverageLatency    time.Duration
	MinLatency        time.Duration
	MaxLatency        time.Duration
	SuccessRate       float64 // Percentage of messages successfully delivered
}

// String returns a string representation of the benchmark result
func (r BenchmarkResult) String() string {
	return fmt.Sprintf(`
Benchmark Results:
-----------------
Messages:            %d
Duration:            %v
Messages per second: %.2f
Average latency:     %v
Min latency:         %v
Max latency:         %v
Success rate:        %.2f%%
`,
		r.TotalMessages,
		r.TotalDuration,
		r.MessagesPerSecond,
		r.AverageLatency,
		r.MinLatency,
		r.MaxLatency,
		r.SuccessRate*100,
	)
}

// RunBenchmark runs a benchmark with the given configuration
func RunBenchmark(ps *PubSub, config BenchmarkConfig) BenchmarkResult {
	// Prepare result
	result := BenchmarkResult{
		Config:      config,
		MinLatency:  time.Hour, // Initialize with a large value
		SuccessRate: 1.0,       // Assume all messages delivered
	}

	// Validate configuration
	if config.MessageCount <= 0 {
		config.MessageCount = 100 // Default to a reasonable value
	}
	if config.MessageSize <= 0 {
		config.MessageSize = 256 // Default to a reasonable value
	}

	// Create a completely new PubSub instance for benchmarking
	// This is important to avoid any interference with existing consumers
	benchmarkPubSub := NewPubSub()

	// Create a channel to collect latency data
	latencyChan := make(chan time.Duration, config.MessageCount)

	// Create mutex for concurrent access to counters
	var mutex sync.Mutex
	received := 0

	// Track processed messages
	processedMessages := make(map[string]bool)

	// Use a channel for completion notification instead of WaitGroup
	done := make(chan struct{})

	// Setup benchmark consumer
	benchConsumer := func(msg model.Message) {
		receiveTime := time.Now()

		// Check if it's a benchmark message (not a warmup message)
		if data, ok := msg.Content.(map[string]interface{}); ok {
			if benchVal, ok := data["benchmark"].(bool); ok && benchVal {
				// Check for warmup vs actual benchmark message
				msgIndexFloat, ok := data["msgIndex"].(float64)
				if !ok {
					// Try to get it as an int if not float64
					msgIndexInt, ok := data["msgIndex"].(int)
					if ok {
						msgIndexFloat = float64(msgIndexInt)
					}
				}

				// Only process actual benchmark messages (non-negative index)
				if ok && msgIndexFloat >= 0 {
					if publishTimeStr, ok := data["publishTime"].(string); ok {
						if publishTime, err := time.Parse(time.RFC3339Nano, publishTimeStr); err == nil {
							latency := receiveTime.Sub(publishTime)

							mutex.Lock()
							// Only count each message once
							if !processedMessages[msg.ID] {
								processedMessages[msg.ID] = true
								received++
								latencyChan <- latency

								// If all messages received, signal completion
								if received >= config.MessageCount {
									select {
									case <-done: // Already closed
									default:
										close(done)
									}
								}
							}
							mutex.Unlock()
						}
					}
				}
			}
		}
	}

	// Add benchmark consumer
	for i := 0; i < config.ConcurrentConsumers; i++ {
		benchmarkPubSub = Subscribe(benchmarkPubSub, benchConsumer)
	}

	// Generate message content of the specified size
	generateMessage := func(i int) map[string]interface{} {
		// Ensure message size is reasonable and positive
		safeSize := config.MessageSize
		if safeSize < 100 {
			safeSize = 100 // Minimum size to accommodate metadata
		}

		// Calculate padding size, ensuring it's positive
		paddingSize := safeSize - 100 // Reserve space for metadata
		if paddingSize <= 0 {
			paddingSize = 1 // At least one byte
		}

		// Create a message with the required size
		padding := make([]byte, paddingSize)
		for j := range padding {
			padding[j] = byte(j % 256)
		}

		return map[string]interface{}{
			"benchmark":   true,
			"msgIndex":    float64(i), // Store as float64 to avoid type issues with JSON
			"publishTime": time.Now().Format(time.RFC3339Nano),
			"padding":     string(padding), // Convert to string for JSON compatibility
		}
	}

	// Use a separate instance for warmup
	fmt.Println("Warming up with", config.WarmupCount, "messages...")
	warmupPubSub := NewPubSub() // Separate instance for warmup

	for i := 0; i < config.WarmupCount; i++ {
		// Negative indexes indicate warmup messages
		warmupMsg := generateMessage(-i - 1)
		Publish(warmupPubSub, warmupMsg)
	}

	// Push all warmup messages
	PushAll(warmupPubSub)

	// Wait for warmup to complete
	time.Sleep(500 * time.Millisecond)

	// Run the actual benchmark
	fmt.Printf("Running benchmark with %d messages, %d producers, %d consumers...\n",
		config.MessageCount, config.ConcurrentProducers, config.ConcurrentConsumers)

	// Reset counters for the actual benchmark
	mutex.Lock()
	received = 0
	processedMessages = make(map[string]bool)
	mutex.Unlock()

	// Create messages and their IDs
	messages := make([]model.Message, config.MessageCount)
	messageIDs := make([]string, config.MessageCount)

	// Start timing
	start := time.Now()

	// Prepare all messages first
	for i := 0; i < config.MessageCount; i++ {
		msgContent := generateMessage(i)
		// Create message directly instead of publishing first
		msg := model.NewMessage(msgContent)
		messages[i] = msg
		messageIDs[i] = msg.ID
		// Store it manually
		benchmarkPubSub.store.AddMessage(msg)
	}

	// Push all messages in the configured pattern
	if config.ConcurrentProducers == 1 {
		// Single producer - push in sequence
		for i := 0; i < config.MessageCount; i++ {
			// Use a direct call without goroutines to ensure sequence
			msg := messages[i]
			for _, consumer := range benchmarkPubSub.consumers {
				consumer(msg)
			}
		}
	} else {
		// Multiple producers
		var producerWg sync.WaitGroup
		producerWg.Add(config.ConcurrentProducers)

		for p := 0; p < config.ConcurrentProducers; p++ {
			go func(producerID int) {
				defer producerWg.Done()

				// Calculate message range for this producer
				messagesPerProducer := config.MessageCount / config.ConcurrentProducers
				startIndex := producerID * messagesPerProducer
				endIndex := startIndex + messagesPerProducer

				// Last producer gets any remaining messages
				if producerID == config.ConcurrentProducers-1 {
					endIndex = config.MessageCount
				}

				// Push messages for this producer
				for i := startIndex; i < endIndex; i++ {
					// Get consumers safely
					benchmarkPubSub.mutex.RLock()
					consumersCopy := make([]ConsumerFunc, len(benchmarkPubSub.consumers))
					copy(consumersCopy, benchmarkPubSub.consumers)
					benchmarkPubSub.mutex.RUnlock()

					// Call each consumer directly
					for _, consumer := range consumersCopy {
						consumer(messages[i])
					}

					// Add a small delay to avoid CPU saturation
					time.Sleep(time.Microsecond)
				}
			}(p)
		}

		// Wait for all producers to finish
		producerWg.Wait()
	}

	// Wait for completion or timeout
	fmt.Println("Waiting for all messages to be processed...")

	select {
	case <-done:
		fmt.Println("All messages processed successfully.")
	case <-time.After(config.RunDuration):
		fmt.Println("Benchmark timeout reached.")
	}

	end := time.Now()
	close(latencyChan)

	// Process results
	result.TotalMessages = received
	result.TotalDuration = end.Sub(start)
	if result.TotalDuration.Seconds() > 0 {
		result.MessagesPerSecond = float64(received) / result.TotalDuration.Seconds()
	}
	if config.MessageCount > 0 {
		result.SuccessRate = float64(received) / float64(config.MessageCount)
	}

	// Process latency data
	var totalLatency time.Duration
	count := 0

	for latency := range latencyChan {
		totalLatency += latency
		count++

		if latency < result.MinLatency {
			result.MinLatency = latency
		}

		if latency > result.MaxLatency {
			result.MaxLatency = latency
		}
	}

	if count > 0 {
		result.AverageLatency = totalLatency / time.Duration(count)
	}

	fmt.Printf("Received %d out of %d messages (%.2f%%)\n",
		received, config.MessageCount, 100.0*float64(received)/float64(config.MessageCount))

	return result
}

// RunFunctionalBenchmark runs a benchmark that tests different message patterns and sizes
func RunFunctionalBenchmark(ps *PubSub) map[string]BenchmarkResult {
	results := make(map[string]BenchmarkResult)

	// Small messages, single producer/consumer
	smallConfig := NewBenchmarkConfig().
		WithMessageCount(1000).
		WithMessageSize(128).
		WithConcurrentProducers(1).
		WithConcurrentConsumers(1).
		WithRunDuration(5 * time.Second)

	results["small_single"] = RunBenchmark(ps, smallConfig)

	// Medium messages, multiple producers/consumers
	mediumConfig := NewBenchmarkConfig().
		WithMessageCount(500).
		WithMessageSize(1024).
		WithConcurrentProducers(2).
		WithConcurrentConsumers(2).
		WithRunDuration(5 * time.Second)

	results["medium_multi"] = RunBenchmark(ps, mediumConfig)

	// Large messages, single producer/consumer
	largeConfig := NewBenchmarkConfig().
		WithMessageCount(100).
		WithMessageSize(10240). // 10KB
		WithConcurrentProducers(1).
		WithConcurrentConsumers(1).
		WithRunDuration(5 * time.Second)

	results["large_single"] = RunBenchmark(ps, largeConfig)

	return results
}

// PrintBenchmarkSummary prints a summary of multiple benchmark results
func PrintBenchmarkSummary(results map[string]BenchmarkResult) {
	fmt.Println("\nBenchmark Summary:")
	fmt.Println("==================")

	for name, result := range results {
		fmt.Printf("\n%s:\n", name)
		fmt.Printf("  Messages:            %d\n", result.TotalMessages)
		fmt.Printf("  Messages per second: %.2f\n", result.MessagesPerSecond)
		fmt.Printf("  Average latency:     %v\n", result.AverageLatency)
		fmt.Printf("  Success rate:        %.2f%%\n", result.SuccessRate*100)
	}
}
