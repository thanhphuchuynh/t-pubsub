package pubsub

import (
	"sync"
	"time"

	"github.com/tphuc/pubsub/model"
)

// ConsumerFunc defines the signature for consumer functions
type ConsumerFunc func(msg model.Message)

// MessageStore is a functional interface for message operations
type MessageStore interface {
	AddMessage(msg model.Message) MessageStore
	GetMessages() []model.Message
	GetMessageByID(id string) (model.Message, bool)
	ReplaceMessage(id string, msg model.Message) MessageStore
	ClearMessages() MessageStore
}

// InMemoryMessageStore implements MessageStore
type InMemoryMessageStore struct {
	messages []model.Message
	mutex    sync.RWMutex
}

// NewInMemoryMessageStore creates a new message store
func NewInMemoryMessageStore() *InMemoryMessageStore {
	return &InMemoryMessageStore{
		messages: make([]model.Message, 0),
	}
}

// AddMessage adds a message to the store
func (store *InMemoryMessageStore) AddMessage(msg model.Message) MessageStore {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	// Create a new slice to avoid modifying the original (immutability)
	newMessages := make([]model.Message, len(store.messages), len(store.messages)+1)
	copy(newMessages, store.messages)
	store.messages = append(newMessages, msg)

	return store
}

// GetMessages returns all messages
func (store *InMemoryMessageStore) GetMessages() []model.Message {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	// Return a copy to maintain immutability
	messagesCopy := make([]model.Message, len(store.messages))
	copy(messagesCopy, store.messages)

	return messagesCopy
}

// GetMessageByID returns a message by its ID
func (store *InMemoryMessageStore) GetMessageByID(id string) (model.Message, bool) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	for _, msg := range store.messages {
		if msg.ID == id {
			return msg, true
		}
	}

	return model.Message{}, false
}

// ReplaceMessage replaces a message with the same ID
func (store *InMemoryMessageStore) ReplaceMessage(id string, msg model.Message) MessageStore {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	for i, existingMsg := range store.messages {
		if existingMsg.ID == id {
			// Create a new slice (immutability)
			newMessages := make([]model.Message, len(store.messages))
			copy(newMessages, store.messages)
			newMessages[i] = msg
			store.messages = newMessages
			return store
		}
	}

	// If not found, add as new
	newMessages := make([]model.Message, len(store.messages), len(store.messages)+1)
	copy(newMessages, store.messages)
	store.messages = append(newMessages, msg)

	return store
}

// ClearMessages clears all messages
func (store *InMemoryMessageStore) ClearMessages() MessageStore {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	store.messages = make([]model.Message, 0)
	return store
}

// PubSub represents the pub/sub service
type PubSub struct {
	store        MessageStore
	consumers    []ConsumerFunc
	mutex        sync.RWMutex
	pushInterval time.Duration
	stopChan     chan struct{}
	metrics      *MetricsRegistry
}

// Option is a functional option for configuring PubSub
type Option func(*PubSub)

// WithPushInterval sets the push interval
func WithPushInterval(interval time.Duration) Option {
	return func(ps *PubSub) {
		ps.pushInterval = interval
	}
}

// WithMessageStore sets the message store
func WithMessageStore(store MessageStore) Option {
	return func(ps *PubSub) {
		ps.store = store
	}
}

// WithMetrics sets custom metrics for the PubSub service
func WithMetrics(metrics *MetricsRegistry) Option {
	return func(ps *PubSub) {
		ps.metrics = metrics
	}
}

// NewPubSub creates a new PubSub service with functional options
func NewPubSub(options ...Option) *PubSub {
	ps := &PubSub{
		store:        NewInMemoryMessageStore(),
		consumers:    make([]ConsumerFunc, 0),
		mutex:        sync.RWMutex{},
		pushInterval: 2 * time.Second, // Default interval
		stopChan:     make(chan struct{}),
		metrics:      GetDefaultMetrics(), // Use default metrics by default
	}

	// Apply all options
	for _, option := range options {
		option(ps)
	}

	return ps
}

// SetPushInterval sets the interval for automatic message pushing (functional style)
func SetPushInterval(ps *PubSub, interval time.Duration) *PubSub {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	// Create a new PubSub with updated interval (immutability)
	newPS := &PubSub{
		store:        ps.store,
		consumers:    ps.consumers,
		mutex:        sync.RWMutex{},
		pushInterval: interval,
		stopChan:     ps.stopChan,
	}

	return newPS
}

// Subscribe adds a new consumer (functional approach)
func Subscribe(ps *PubSub, consumer ConsumerFunc) *PubSub {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	// Create a copy of consumers slice (immutability)
	newConsumers := make([]ConsumerFunc, len(ps.consumers), len(ps.consumers)+1)
	copy(newConsumers, ps.consumers)

	// Add the new consumer
	newConsumers = append(newConsumers, consumer)

	// Create a new PubSub with updated consumers
	newPS := &PubSub{
		store:        ps.store,
		consumers:    newConsumers,
		mutex:        sync.RWMutex{},
		pushInterval: ps.pushInterval,
		stopChan:     ps.stopChan,
		metrics:      ps.metrics,
	}

	// Update consumer count metric
	newPS.metrics.ConsumerCount.Set(float64(len(newConsumers)))

	return newPS
}

// Publish adds a message to the queue (as a method for backwards compatibility)
func (ps *PubSub) Publish(content interface{}) string {
	return Publish(ps, content)
}

// Publish adds a message to the queue (functional approach)
func Publish(ps *PubSub, content interface{}) string {
	msg := model.NewMessage(content)
	ps.store.AddMessage(msg)

	// Update metrics
	ps.metrics.MessagesPublished.Inc()
	ps.metrics.QueueSize.Inc()

	// Estimate message size
	size := estimateMessageSize(content)
	ps.metrics.MessageSize.Observe(float64(size))

	return msg.ID
}

// PublishWithID adds a message with a custom ID (as a method for backwards compatibility)
func (ps *PubSub) PublishWithID(id string, content interface{}) string {
	return PublishWithID(ps, id, content)
}

// PublishWithID adds a message with a custom ID (functional approach)
func PublishWithID(ps *PubSub, id string, content interface{}) string {
	msg := model.NewMessageWithID(id, content)
	ps.store.ReplaceMessage(id, msg)

	// Update metrics
	ps.metrics.MessagesPublished.Inc()
	ps.metrics.QueueSize.Inc()

	// Estimate message size
	size := estimateMessageSize(content)
	ps.metrics.MessageSize.Observe(float64(size))

	return msg.ID
}

// PushByID pushes a specific message to all consumers (as a method for backwards compatibility)
func (ps *PubSub) PushByID(messageID string) bool {
	return PushByID(ps, messageID)
}

// PushByID pushes a specific message to all consumers (functional approach)
func PushByID(ps *PubSub, messageID string) bool {
	msg, found := ps.store.GetMessageByID(messageID)
	if !found {
		return false
	}

	// Get a copy of consumers
	ps.mutex.RLock()
	consumers := make([]ConsumerFunc, len(ps.consumers))
	copy(consumers, ps.consumers)
	ps.mutex.RUnlock()

	// Apply each consumer to the message
	start := time.Now()
	for _, consumer := range consumers {
		// Use a goroutine to avoid blocking
		go func(c ConsumerFunc, m model.Message, startTime time.Time) {
			c(m)
			ps.metrics.MessagesDelivered.Inc()
			ps.metrics.ConsumerLatency.Observe(time.Since(startTime).Seconds())
		}(consumer, msg, start)
	}

	return true
}

// PushAll pushes all messages to all consumers (as a method for backwards compatibility)
func (ps *PubSub) PushAll() {
	PushAll(ps)
}

// PushAll pushes all messages to all consumers (functional approach)
func PushAll(ps *PubSub) {
	messages := ps.store.GetMessages()

	// Update queue size metric
	ps.metrics.QueueSize.Set(float64(len(messages)))

	// Get a copy of consumers
	ps.mutex.RLock()
	consumers := make([]ConsumerFunc, len(ps.consumers))
	copy(consumers, ps.consumers)
	ps.mutex.RUnlock()

	// For each message, apply all consumers
	start := time.Now()
	for _, msg := range messages {
		for _, consumer := range consumers {
			// Use a goroutine to avoid blocking
			go func(c ConsumerFunc, m model.Message, startTime time.Time) {
				c(m)
				ps.metrics.MessagesDelivered.Inc()
				ps.metrics.ConsumerLatency.Observe(time.Since(startTime).Seconds())
			}(consumer, msg, start)
		}
	}
}

// StartAutoPush starts pushing messages automatically at intervals (as a method for backwards compatibility)
func (ps *PubSub) StartAutoPush() *PubSub {
	return StartAutoPush(ps)
}

// StartAutoPush starts pushing messages automatically at intervals (functional approach)
func StartAutoPush(ps *PubSub) *PubSub {
	ps.mutex.Lock()
	ps.stopChan = make(chan struct{})
	ps.mutex.Unlock()

	go func() {
		ticker := time.NewTicker(ps.pushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				PushAll(ps)
			case <-ps.stopChan:
				return
			}
		}
	}()

	return ps
}

// StopAutoPush stops the automatic pushing of messages (as a method for backwards compatibility)
func (ps *PubSub) StopAutoPush() *PubSub {
	return StopAutoPush(ps)
}

// StopAutoPush stops the automatic pushing of messages (functional approach)
func StopAutoPush(ps *PubSub) *PubSub {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if ps.stopChan != nil {
		close(ps.stopChan)
	}

	return ps
}

// Clear removes all messages from the queue (as a method for backwards compatibility)
func (ps *PubSub) Clear() *PubSub {
	return Clear(ps)
}

// Clear removes all messages from the queue (functional approach)
func Clear(ps *PubSub) *PubSub {
	ps.store.ClearMessages()
	ps.metrics.QueueSize.Set(0)
	return ps
}

// For backward compatibility, keep these method aliases
func (ps *PubSub) SetPushInterval(interval time.Duration) *PubSub {
	return SetPushInterval(ps, interval)
}

func (ps *PubSub) Subscribe(consumer ConsumerFunc) *PubSub {
	return Subscribe(ps, consumer)
}

// SubscribeBatchConsumer adds a batch consumer to the pubsub system
func SubscribeBatchConsumer(ps *PubSub, config BatchConsumerConfig, processor func([]model.Message)) *PubSub {
	batchConsumer := NewBatchConsumer(config, processor)
	return Subscribe(ps, batchConsumer.AsConsumerFunc())
}

// SubscribeBatchConsumer adds a batch consumer (method for backward compatibility)
func (ps *PubSub) SubscribeBatchConsumer(config BatchConsumerConfig, processor func([]model.Message)) *PubSub {
	return SubscribeBatchConsumer(ps, config, processor)
}

// Helper function to estimate message size
func estimateMessageSize(content interface{}) int {
	switch v := content.(type) {
	case string:
		return len(v)
	case []byte:
		return len(v)
	case map[string]interface{}:
		size := 0
		for k, val := range v {
			size += len(k) + estimateMessageSize(val)
		}
		return size
	case map[string]string:
		size := 0
		for k, val := range v {
			size += len(k) + len(val)
		}
		return size
	default:
		return 100 // default size estimate
	}
}

// GetMetrics returns the metrics registry for this PubSub instance
func (ps *PubSub) GetMetrics() *MetricsRegistry {
	return ps.metrics
}

// GetConsumers returns a copy of the consumers slice
func (ps *PubSub) GetConsumers() []ConsumerFunc {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	consumers := make([]ConsumerFunc, len(ps.consumers))
	copy(consumers, ps.consumers)
	return consumers
}
