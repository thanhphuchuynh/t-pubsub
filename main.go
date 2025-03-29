package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/thanhphuchuynh/t-pubsub/model"
	"github.com/thanhphuchuynh/t-pubsub/pubsub"
)

var (
	// Global PubSub instance for API access
	globalPS       *pubsub.PubSub
	simulationDone chan struct{}
	isSimulating   bool
)

func main() {
	// Start Prometheus metrics server and HTTP API
	go startServer()

	// Create a new PubSub service with functional options
	globalPS = pubsub.NewPubSub(
		pubsub.WithPushInterval(2 * time.Second),
	)

	// Create a reporter for system metrics
	reporter := pubsub.NewReporter()
	reporter.Start()

	// Consumer functions
	consumerOne := func(msg model.Message) {
		fmt.Printf("Consumer 1 received: %v (ID: %s)\n", msg.Content, msg.ID)
	}

	consumerTwo := func(msg model.Message) {
		fmt.Printf("Consumer 2 received: %v (ID: %s)\n", msg.Content, msg.ID)
	}

	// Add consumers using functional approach with reporting
	globalPS = pubsub.SubscribeWithReporting(globalPS, reporter, consumerOne)
	globalPS = pubsub.SubscribeWithReporting(globalPS, reporter, consumerTwo)

	// Add a batch consumer with batch size 2 and timeout 3 seconds
	batchConfig := pubsub.NewBatchConsumerConfig().
		WithBatchSize(2).
		WithTimeout(3 * time.Second)

	batchProcessor := func(batch []model.Message) {
		fmt.Printf("\nBatch Processor received %d messages:\n", len(batch))
		for i, msg := range batch {
			fmt.Printf("  %d. Message: %v (ID: %s)\n", i+1, msg.Content, msg.ID)
		}
		fmt.Println()
	}

	globalPS = pubsub.SubscribeBatchConsumer(globalPS, batchConfig, batchProcessor)

	// Basic demo starts here
	runBasicDemo(globalPS)

	// Generate and print the report
	fmt.Println("\n--- System Report ---")
	reporter.PrintReport(os.Stdout)

	// Save the report to a file
	err := reporter.SaveReportToFile("pubsub_report.json")
	if err != nil {
		fmt.Println("Error saving report:", err)
	} else {
		fmt.Println("Report saved to pubsub_report.json")
	}

	fmt.Println("\nAPI server is running at http://localhost:2112")
	fmt.Println("Available endpoints:")
	fmt.Println("- GET /metrics - Prometheus metrics")
	fmt.Println("- POST /benchmark - Run benchmark with optional parameters")
	fmt.Println("- POST /simulate/start - Start continuous message simulation")
	fmt.Println("- POST /simulate/stop - Stop continuous simulation")
	fmt.Println("- GET /status - Get system status")
	fmt.Println("\nPress Ctrl+C to exit.")

	// Keep the application running to continue serving metrics and API
	select {}
}

func runBasicDemo(ps *pubsub.PubSub) {
	// Publish some messages
	id1 := pubsub.Publish(ps, "Hello, World!")
	id2 := pubsub.Publish(ps, 42)
	id3 := pubsub.Publish(ps, map[string]string{"name": "John", "role": "Developer"})

	fmt.Println("Published messages with IDs:", id1, id2, id3)
	fmt.Println("Prometheus metrics available at http://localhost:2112/metrics")

	// Publish message with custom ID using functional approach
	customID := "custom-message-2023"
	pubsub.PublishWithID(ps, customID, "This message has a custom ID")
	fmt.Println("Published message with custom ID:", customID)

	// Demonstrating batch consumer with full batch
	fmt.Println("\n--- Demonstrating batch consumer with full batch ---")
	batchMsg1 := pubsub.Publish(ps, "Batch message 1")
	fmt.Println("Published batch message 1:", batchMsg1)
	time.Sleep(1 * time.Second)

	batchMsg2 := pubsub.Publish(ps, "Batch message 2")
	fmt.Println("Published batch message 2:", batchMsg2)

	// Push these messages to all consumers including the batch consumer
	pubsub.PushByID(ps, batchMsg1)
	pubsub.PushByID(ps, batchMsg2)

	// Wait to ensure batch processing completes
	time.Sleep(2 * time.Second)

	// Demonstrating batch consumer with timeout
	fmt.Println("\n--- Demonstrating batch consumer with timeout ---")
	timeoutMsg := pubsub.Publish(ps, "Timeout message")
	fmt.Println("Published timeout message:", timeoutMsg)

	// Push the message to all consumers including the batch consumer
	pubsub.PushByID(ps, timeoutMsg)

	// Wait for timeout to occur
	fmt.Println("Waiting for batch timeout (3 seconds)...")
	time.Sleep(4 * time.Second)

	fmt.Println("\nBasic demo completed. API server is ready for requests.")
}

// askForFullBenchmark asks the user if they want to run the full benchmark suite
func askForFullBenchmark() bool {
	fmt.Print("Run full benchmark suite? (y/n): ")
	var response string
	fmt.Scanln(&response)
	return response == "y" || response == "Y"
}

// startServer starts HTTP server for metrics and API
func startServer() {
	// Set up the routes
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Prometheus metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	// Benchmark API endpoint
	http.HandleFunc("/benchmark", handleBenchmark)

	// Simulation API endpoints
	http.HandleFunc("/simulate/start", handleStartSimulation)
	http.HandleFunc("/simulate/stop", handleStopSimulation)

	// Status endpoint
	http.HandleFunc("/status", handleStatus)

	// Start server with simple multiplexer
	log.Println("Starting server on :2112")
	if err := http.ListenAndServe(":2112", nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

// BenchmarkRequest represents the request body for the benchmark API
type BenchmarkRequest struct {
	MessageCount        int    `json:"messageCount"`
	ConcurrentProducers int    `json:"concurrentProducers"`
	ConcurrentConsumers int    `json:"concurrentConsumers"`
	MessageSize         int    `json:"messageSize"`
	WarmupCount         int    `json:"warmupCount"`
	RunDuration         string `json:"runDuration"` // Duration in string format (e.g., "10s", "1m")
}

// SimulationConfig represents configuration for continuous simulation
type SimulationConfig struct {
	MessageRate int    `json:"messageRate"` // Messages per second
	MessageSize int    `json:"messageSize"` // Message size in bytes
	Duration    string `json:"duration"`    // Optional duration (e.g., "1h")
	ConsumerLag string `json:"consumerLag"` // Optional consumer lag simulation (e.g., "100ms")
}

// SystemStatus represents the current system status
type SystemStatus struct {
	IsSimulating      bool      `json:"isSimulating"`
	SimulationStarted time.Time `json:"simulationStarted,omitempty"`
	MessagesPublished int       `json:"messagesPublished"`
	MessagesDelivered int       `json:"messagesDelivered"`
	ConsumerCount     int       `json:"consumerCount"`
	QueueSize         int       `json:"queueSize"`
}

// handleBenchmark processes the benchmark API requests
func handleBenchmark(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req BenchmarkRequest
	var benchConfig pubsub.BenchmarkConfig

	// Set default values
	benchConfig = pubsub.NewBenchmarkConfig()

	// Try to decode JSON request body
	if r.Header.Get("Content-Type") == "application/json" {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&req); err == nil {
			// Apply custom configuration if provided
			if req.MessageCount > 0 {
				benchConfig = benchConfig.WithMessageCount(req.MessageCount)
			}
			if req.ConcurrentProducers > 0 {
				benchConfig = benchConfig.WithConcurrentProducers(req.ConcurrentProducers)
			}
			if req.ConcurrentConsumers > 0 {
				benchConfig = benchConfig.WithConcurrentConsumers(req.ConcurrentConsumers)
			}
			if req.MessageSize > 0 {
				benchConfig = benchConfig.WithMessageSize(req.MessageSize)
			}
			if req.WarmupCount > 0 {
				benchConfig = benchConfig.WithWarmupCount(req.WarmupCount)
			}
			if req.RunDuration != "" {
				if duration, err := time.ParseDuration(req.RunDuration); err == nil {
					benchConfig = benchConfig.WithRunDuration(duration)
				}
			}
		}
	} else {
		// Try to parse form or query parameters
		if r.ParseForm() == nil {
			if count, err := strconv.Atoi(r.FormValue("messageCount")); err == nil && count > 0 {
				benchConfig = benchConfig.WithMessageCount(count)
			}
			if count, err := strconv.Atoi(r.FormValue("concurrentProducers")); err == nil && count > 0 {
				benchConfig = benchConfig.WithConcurrentProducers(count)
			}
			if count, err := strconv.Atoi(r.FormValue("concurrentConsumers")); err == nil && count > 0 {
				benchConfig = benchConfig.WithConcurrentConsumers(count)
			}
			if size, err := strconv.Atoi(r.FormValue("messageSize")); err == nil && size > 0 {
				benchConfig = benchConfig.WithMessageSize(size)
			}
			if count, err := strconv.Atoi(r.FormValue("warmupCount")); err == nil && count > 0 {
				benchConfig = benchConfig.WithWarmupCount(count)
			}
			if dur := r.FormValue("runDuration"); dur != "" {
				if duration, err := time.ParseDuration(dur); err == nil {
					benchConfig = benchConfig.WithRunDuration(duration)
				}
			}
		}
	}

	// Create a separate PubSub instance for benchmarking
	benchPS := pubsub.NewPubSub(
		pubsub.WithPushInterval(100 * time.Millisecond),
	)

	// Run the benchmark
	fmt.Printf("Running benchmark with config: %+v\n", benchConfig)
	result := pubsub.RunBenchmark(benchPS, benchConfig)

	// Return the result as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleStartSimulation starts a continuous message simulation
func handleStartSimulation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// If already simulating, return error
	if isSimulating {
		http.Error(w, "Simulation already running", http.StatusConflict)
		return
	}

	// Parse simulation config
	var config SimulationConfig
	config.MessageRate = 10   // Default: 10 messages per second
	config.MessageSize = 1024 // Default: 1KB messages

	// Try to decode JSON request body
	if r.Header.Get("Content-Type") == "application/json" {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&config); err != nil {
			// If error, just use defaults
			fmt.Printf("Error parsing simulation config: %v, using defaults\n", err)
		}
	} else {
		// Try to parse form or query parameters
		if r.ParseForm() == nil {
			if rate, err := strconv.Atoi(r.FormValue("messageRate")); err == nil && rate > 0 {
				config.MessageRate = rate
			}
			if size, err := strconv.Atoi(r.FormValue("messageSize")); err == nil && size > 0 {
				config.MessageSize = size
			}
			config.Duration = r.FormValue("duration")
			config.ConsumerLag = r.FormValue("consumerLag")
		}
	}

	// Setup simulation done channel
	simulationDone = make(chan struct{})
	isSimulating = true

	// Parse optional duration
	var duration time.Duration
	var hasTimeout bool
	if config.Duration != "" {
		if parsedDuration, err := time.ParseDuration(config.Duration); err == nil {
			duration = parsedDuration
			hasTimeout = true
		}
	}

	// Parse optional consumer lag
	var consumerLag time.Duration
	if config.ConsumerLag != "" {
		if parsedLag, err := time.ParseDuration(config.ConsumerLag); err == nil {
			consumerLag = parsedLag
		}
	}

	// Start simulation in background
	go runContinuousSimulation(config.MessageRate, config.MessageSize, consumerLag)

	// If duration specified, setup timeout
	if hasTimeout {
		go func() {
			select {
			case <-time.After(duration):
				handleSimulationStop()
			case <-simulationDone:
				return // Simulation was stopped manually
			}
		}()
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "started",
		"messageRate": config.MessageRate,
		"messageSize": config.MessageSize,
		"duration":    config.Duration,
	})
}

// handleStopSimulation stops an ongoing simulation
func handleStopSimulation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !isSimulating {
		http.Error(w, "No simulation is running", http.StatusNotFound)
		return
	}

	handleSimulationStop()

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "stopped",
	})
}

// handleStatus returns the current system status
func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get metrics from global instance
	metrics := globalPS.GetMetrics()

	// Create status response
	status := SystemStatus{
		IsSimulating:      isSimulating,
		ConsumerCount:     len(globalPS.GetConsumers()), // Use GetConsumers directly
		MessagesPublished: int(getMetricValue(metrics.MessagesPublished)),
		MessagesDelivered: int(getMetricValue(metrics.MessagesDelivered)),
		QueueSize:         int(getMetricValue(metrics.QueueSize)),
	}

	// Return as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Helper function to extract a value from a Prometheus metric
func getMetricValue(metric interface{}) float64 {
	// Use type switches instead of assertions
	switch m := metric.(type) {
	case pubsub.MetricGetter:
		return m.GetValue()
	default:
		return 0
	}
}

// runContinuousSimulation runs a continuous simulation with the given parameters
func runContinuousSimulation(messagesPerSecond, messageSize int, consumerLag time.Duration) {
	fmt.Printf("Starting continuous simulation: %d msgs/sec, %d bytes/msg\n",
		messagesPerSecond, messageSize)

	// Calculate delay between messages
	delay := time.Second / time.Duration(messagesPerSecond)

	// Create a ticker for regular message publishing
	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	counter := 0

	for {
		select {
		case <-ticker.C:
			// Create message content with specified size
			msgContent := generateSimulationMessage(counter, messageSize)

			// Publish to the global PubSub instance
			msgID := pubsub.Publish(globalPS, msgContent)

			// If consumer lag is specified, delay the push
			if consumerLag > 0 {
				go func(id string, lag time.Duration) {
					time.Sleep(lag)
					pubsub.PushByID(globalPS, id)
				}(msgID, consumerLag)
			} else {
				// Otherwise push immediately
				pubsub.PushByID(globalPS, msgID)
			}

			counter++

		case <-simulationDone:
			fmt.Println("Simulation stopped")
			return
		}
	}
}

// handleSimulationStop stops the simulation
func handleSimulationStop() {
	if isSimulating {
		close(simulationDone)
		isSimulating = false
		fmt.Println("Simulation stopped")
	}
}

// generateSimulationMessage generates a message of the specified size
func generateSimulationMessage(index, size int) map[string]interface{} {
	// Create padding to reach the desired size
	paddingSize := size - 100 // Approximate size for metadata
	if paddingSize < 0 {
		paddingSize = 1
	}

	padding := make([]byte, paddingSize)
	for i := range padding {
		padding[i] = byte(i % 256)
	}

	return map[string]interface{}{
		"index":      index,
		"timestamp":  time.Now().Format(time.RFC3339Nano),
		"simulation": true,
		"padding":    string(padding),
	}
}
