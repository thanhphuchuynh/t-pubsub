package pubsub

import (
	"sync"
	"time"

	"github.com/tphuc/pubsub/model"
)

// BatchConsumerConfig defines configuration for batch consumer
type BatchConsumerConfig struct {
	BatchSize int           // Number of messages to collect before processing
	Timeout   time.Duration // Maximum time to wait for batch to fill
}

// NewBatchConsumerConfig creates a new batch consumer config with defaults
func NewBatchConsumerConfig() BatchConsumerConfig {
	return BatchConsumerConfig{
		BatchSize: 2,           // Default batch size
		Timeout:   time.Second, // Default timeout
	}
}

// WithBatchSize sets the batch size
func (c BatchConsumerConfig) WithBatchSize(size int) BatchConsumerConfig {
	c.BatchSize = size
	return c
}

// WithTimeout sets the timeout duration
func (c BatchConsumerConfig) WithTimeout(timeout time.Duration) BatchConsumerConfig {
	c.Timeout = timeout
	return c
}

// BatchConsumer represents a consumer that processes messages in batches
type BatchConsumer struct {
	config     BatchConsumerConfig
	processor  func([]model.Message)
	messages   []model.Message
	mutex      sync.Mutex
	timer      *time.Timer
	processing bool
	metrics    *MetricsRegistry
}

// NewBatchConsumer creates a new batch consumer
func NewBatchConsumer(config BatchConsumerConfig, processor func([]model.Message)) *BatchConsumer {
	return &BatchConsumer{
		config:     config,
		processor:  processor,
		messages:   make([]model.Message, 0, config.BatchSize),
		mutex:      sync.Mutex{},
		processing: false,
		metrics:    GetDefaultMetrics(),
	}
}

// WithMetrics sets custom metrics for the batch consumer
func (bc *BatchConsumer) WithMetrics(metrics *MetricsRegistry) *BatchConsumer {
	bc.metrics = metrics
	return bc
}

// HandleMessage handles a single message
func (bc *BatchConsumer) HandleMessage(msg model.Message) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	// Add message to batch
	bc.messages = append(bc.messages, msg)

	// If this is the first message in the batch, start the timer
	if len(bc.messages) == 1 && !bc.processing {
		bc.startTimer()
	}

	// If batch is full, process it immediately
	if len(bc.messages) >= bc.config.BatchSize {
		bc.processCurrentBatch()
	}
}

// startTimer starts the timeout timer
func (bc *BatchConsumer) startTimer() {
	bc.processing = true
	bc.timer = time.AfterFunc(bc.config.Timeout, func() {
		bc.mutex.Lock()
		defer bc.mutex.Unlock()

		// If we still have messages to process when the timer fires
		if len(bc.messages) > 0 {
			bc.processCurrentBatch()
		} else {
			bc.processing = false
		}
	})
}

// processCurrentBatch processes the current batch of messages
func (bc *BatchConsumer) processCurrentBatch() {
	if bc.timer != nil {
		bc.timer.Stop()
	}

	// Make a copy of the current batch
	batch := make([]model.Message, len(bc.messages))
	copy(batch, bc.messages)

	// Record batch size in metrics
	bc.metrics.BatchSize.Observe(float64(len(batch)))

	// Clear the current batch
	bc.messages = make([]model.Message, 0, bc.config.BatchSize)

	// Reset processing flag
	bc.processing = false

	// Process the batch asynchronously
	start := time.Now()
	go func() {
		bc.processor(batch)
		bc.metrics.ConsumerLatency.Observe(time.Since(start).Seconds())

		// Count each message in the batch as delivered
		for range batch {
			bc.metrics.MessagesDelivered.Inc()
		}
	}()
}

// AsConsumerFunc returns a ConsumerFunc that can be used with PubSub
func (bc *BatchConsumer) AsConsumerFunc() ConsumerFunc {
	return func(msg model.Message) {
		bc.HandleMessage(msg)
	}
}
