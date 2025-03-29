package pubsub

import (
	"sync"
	"testing"
	"time"

	"github.com/thanhphuchuynh/t-pubsub/model"
)

func TestBatchConsumerConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		config := NewBatchConsumerConfig()
		if config.BatchSize != 2 {
			t.Errorf("Expected default batch size 2, got %d", config.BatchSize)
		}
		if config.Timeout != time.Second {
			t.Errorf("Expected default timeout 1s, got %v", config.Timeout)
		}
	})

	t.Run("custom config", func(t *testing.T) {
		config := NewBatchConsumerConfig().
			WithBatchSize(5).
			WithTimeout(2 * time.Second)

		if config.BatchSize != 5 {
			t.Errorf("Expected batch size 5, got %d", config.BatchSize)
		}
		if config.Timeout != 2*time.Second {
			t.Errorf("Expected timeout 2s, got %v", config.Timeout)
		}
	})
}

func TestBatchConsumerProcessing(t *testing.T) {
	var processed []model.Message
	var mu sync.Mutex

	processor := func(msgs []model.Message) {
		mu.Lock()
		processed = append(processed, msgs...)
		mu.Unlock()
	}

	t.Run("process on batch size", func(t *testing.T) {
		processed = nil
		config := NewBatchConsumerConfig().WithBatchSize(2)
		consumer := NewBatchConsumer(config, processor)

		msg1 := model.NewMessage("test1")
		msg2 := model.NewMessage("test2")

		consumer.HandleMessage(msg1)
		if len(processed) != 0 {
			t.Error("Messages should not be processed before batch is full")
		}

		consumer.HandleMessage(msg2)
		time.Sleep(100 * time.Millisecond) // Wait for async processing

		mu.Lock()
		if len(processed) != 2 {
			t.Errorf("Expected 2 processed messages, got %d", len(processed))
		}
		if processed[0].Content != "test1" || processed[1].Content != "test2" {
			t.Error("Messages processed in wrong order")
		}
		mu.Unlock()
	})

	t.Run("process on timeout", func(t *testing.T) {
		processed = nil
		config := NewBatchConsumerConfig().
			WithBatchSize(3).
			WithTimeout(100 * time.Millisecond)
		consumer := NewBatchConsumer(config, processor)

		msg := model.NewMessage("timeout test")
		consumer.HandleMessage(msg)

		time.Sleep(200 * time.Millisecond) // Wait for timeout

		mu.Lock()
		if len(processed) != 1 {
			t.Errorf("Expected 1 processed message after timeout, got %d", len(processed))
		}
		if processed[0].Content != "timeout test" {
			t.Error("Wrong message processed")
		}
		mu.Unlock()
	})
}

func TestBatchConsumerIntegration(t *testing.T) {
	var processedBatches [][]model.Message
	var mu sync.Mutex
	done := make(chan bool)

	processor := func(msgs []model.Message) {
		mu.Lock()
		processedBatches = append(processedBatches, msgs)
		mu.Unlock()
		if len(processedBatches) == 2 {
			done <- true // Signal that all messages have been processed
		}
	}

	config := NewBatchConsumerConfig().
		WithBatchSize(2).
		WithTimeout(100 * time.Millisecond)

	ps := NewPubSub() // Enable FIFO mode
	ps = ps.SubscribeBatchConsumer(config, processor)

	// Publish messages
	msg1ID := ps.Publish("test1")
	msg2ID := ps.Publish("test2")
	msg3ID := ps.Publish("test3")

	// Push messages
	ps.PushByID(msg1ID)
	ps.PushByID(msg2ID)
	ps.PushByID(msg3ID)

	time.Sleep(200 * time.Millisecond) // Wait for timeout

	mu.Lock()
	defer mu.Unlock()

	if len(processedBatches) != 2 {
		t.Errorf("Expected 2 batches, got %d", len(processedBatches))
		return
	}

	// First batch should have 2 messages
	if len(processedBatches[0]) != 2 {
		t.Errorf("Expected first batch size 2, got %d", len(processedBatches[0]))
	}

	// Second batch should have 1 message
	if len(processedBatches[1]) != 1 {
		t.Errorf("Expected second batch size 1, got %d", len(processedBatches[1]))
	}

	// Verify message contents
	expectedContents := []string{"test1", "test2"}

	for i, msg := range processedBatches[0] {
		if msg.Content != expectedContents[i] {
			t.Errorf("Expected content %s, got %v", expectedContents[i], msg.Content)
		}
	}

	if processedBatches[1][0].Content != "test3" {
		t.Errorf("Expected content test3, got %v", processedBatches[1][0].Content)
	}
}

func TestBatchConsumerMetrics(t *testing.T) {
	metrics := GetDefaultMetrics()
	processor := func(msgs []model.Message) {
		time.Sleep(10 * time.Millisecond) // Simulate processing time
	}

	config := NewBatchConsumerConfig().
		WithBatchSize(2).
		WithTimeout(100 * time.Millisecond)

	consumer := NewBatchConsumer(config, processor).
		WithMetrics(metrics)

	// Send test messages
	msg1 := model.NewMessage("test1")
	msg2 := model.NewMessage("test2")

	// Get initial delivered messages count
	initialDelivered := metrics.MessagesDelivered.GetValue()

	consumer.HandleMessage(msg1)
	consumer.HandleMessage(msg2)

	time.Sleep(50 * time.Millisecond) // Wait for processing

	// Verify metrics were updated
	newDelivered := metrics.MessagesDelivered.GetValue()
	if newDelivered <= initialDelivered {
		t.Errorf("MessagesDelivered metric was not incremented (before: %f, after: %f)", initialDelivered, newDelivered)
	}

	// Note: We don't test BatchSize directly as it's a Histogram
	// Instead, we can verify that messages were processed in a batch by checking MessagesDelivered
	if newDelivered != initialDelivered+2 {
		t.Errorf("Expected %f messages delivered, got %f", initialDelivered+2, newDelivered)
	}
}
