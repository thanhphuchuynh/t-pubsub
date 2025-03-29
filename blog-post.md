# Building a High-Performance PubSub System with Functional Go

As a software engineer passionate about building robust distributed systems, I'd like to share my experience creating `t-pubsub`, a high-performance publish-subscribe system built with Go using functional programming principles. This project combines modern Go practices with functional programming to create a reliable, metrics-enabled message delivery system.

## Why Another PubSub System?

Most PubSub implementations use object-oriented patterns with mutable state, leading to potential race conditions and hard-to-track bugs in concurrent scenarios. `t-pubsub` takes a different approach by embracing functional programming principles:

- **Immutability**: Operations return new instances instead of modifying existing state
- **Pure Functions**: Minimize side effects for predictable behavior
- **Higher-Order Functions**: Functions that operate on other functions for flexibility

## Core Architecture

The system is built around several key components:

```go
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
```

The core `PubSub` type encapsulates the message store and consumer functions while maintaining thread safety:

```go
type PubSub struct {
    store        MessageStore
    consumers    []ConsumerFunc
    mutex        sync.RWMutex
    pushInterval time.Duration
    stopChan     chan struct{}
    metrics      *MetricsRegistry
}
```

## Functional Design Patterns

### 1. Immutable State Management

Instead of modifying state directly, operations return new instances:

```go
// Subscribe adds a new consumer (functional approach)
func Subscribe(ps *PubSub, consumer ConsumerFunc) *PubSub {
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
    
    return newPS
}
```

### 2. Functional Options Pattern

Configuration is handled through functional options, making it flexible and extensible:

```go
// Create PubSub with custom options
ps := pubsub.NewPubSub(
    pubsub.WithPushInterval(100 * time.Millisecond),
    pubsub.WithMessageStore(customStore),
    pubsub.WithMetrics(customMetrics),
)
```

## Real-Time Performance Monitoring

One of the standout features is comprehensive metrics collection integrated with Prometheus and Grafana. The system tracks:

- Message rates (published/delivered)
- Queue size and utilization
- Consumer latency percentiles
- Processing time distributions
- Error rates

```go
// Metrics are updated automatically during operations
ps.metrics.MessagesPublished.Inc()
ps.metrics.QueueSize.Inc()
ps.metrics.MessagesInQueue.Inc()
ps.metrics.QueueUtilization.Set(float64(len(queueSize)) / float64(1000) * 100)
```

## Batch Processing Capabilities

For high-throughput scenarios, the system supports batch processing:

```go
batchConfig := pubsub.NewBatchConsumerConfig().
    WithBatchSize(10).
    WithTimeout(5 * time.Second)

batchProcessor := func(batch []model.Message) {
    fmt.Printf("Processing batch of %d messages\n", len(batch))
}

ps = pubsub.SubscribeBatchConsumer(ps, batchConfig, batchProcessor)
```

## Performance Tuning

The system includes built-in benchmarking tools for performance optimization:

```bash
curl -X POST http://localhost:2112/benchmark \
  -H "Content-Type: application/json" \
  -d '{
    "messageCount": 1000,
    "concurrentProducers": 4,
    "messageSize": 512
  }'
```

You can also run continuous simulations to test different scenarios:

```bash
curl -X POST http://localhost:2112/simulate/start \
  -H "Content-Type: application/json" \
  -d '{
    "messageRate": 50,
    "numberProducers": 2,
    "numberConsumers": 3,
    "duration": "10m"
  }'
```

## Containerized Deployment

The system is Docker-ready with a complete monitoring stack:

1. PubSub Service: The main application
2. Prometheus: Metrics collection
3. Grafana: Real-time visualization

The included Grafana dashboards provide instant visibility into:
- Message throughput
- Queue metrics
- Consumer performance
- System health indicators

## Conclusion

Building `t-pubsub` with functional programming principles in Go has resulted in a system that is:
- Thread-safe by design
- Easy to reason about
- Highly testable
- Performance-optimized
- Ready for production use

The combination of functional patterns with Go's strong concurrency support creates a robust foundation for building distributed messaging systems. The integrated metrics and monitoring capabilities make it suitable for production environments where observability is crucial.

You can find the complete source code on [GitHub](https://github.com/thanhphuchuynh/t-pubsub).

---

**Tags**: #Go #PubSub #FunctionalProgramming #Monitoring #DistributedSystems
