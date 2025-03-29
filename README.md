# Functional PubSub System

A high-performance, metrics-enabled publish-subscribe system built in Go with functional programming style.

## Features

- **Pure Functional Style**: Immutable data structures and side-effect free operations
- **Flexible Message Handling**: Support for various message types and batch processing
- **Real-time Metrics**: Prometheus integration with Grafana dashboards
- **Benchmarking Tools**: Built-in performance testing and simulation
- **Docker Integration**: Ready to run in containerized environments
- **REST API**: HTTP endpoints for system management and monitoring

## Getting Started

### Prerequisites

- Go 1.18 or higher
- Docker and Docker Compose (for containerized setup)

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/thanhphuchuynh/t-pubsub.git
   cd pubsub
   ```

2. Build and run locally:
   ```bash
   go run main.go
   ```

3. Or use Docker Compose:
   ```bash
   ./run.sh
   ```

## Architecture

This PubSub system is built with functional programming principles:

- **Immutability**: All operations return new instances instead of modifying existing ones
- **Pure Functions**: Minimize side effects and promote predictable behavior
- **Higher-Order Functions**: Functions that take functions as parameters or return functions

The main components are:

- **PubSub Service**: Core message management
- **Message Store**: In-memory storage with thread-safe operations
- **Consumers**: Functions that process messages
- **Batch Consumers**: Specialized consumers that process messages in batches
- **Metrics Registry**: Prometheus metrics collection
- **Reporter**: System performance analysis

## API Endpoints

The system exposes several REST endpoints on port 2112:

### Health and Status

- `GET /health`: System health check
- `GET /status`: Current system status including message counts and consumer information

### Benchmarking

- `POST /benchmark`: Run a benchmark with configurable parameters
  ```bash
  curl -X POST http://localhost:2112/benchmark \
    -H "Content-Type: application/json" \
    -d '{"messageCount": 1000, "concurrentProducers": 4, "messageSize": 512}'
  ```

### Simulation

- `POST /simulate/start`: Start a continuous message simulation
  ```bash
  curl -X POST http://localhost:2112/simulate/start \
    -H "Content-Type: application/json" \
    -d '{
      "messageRate": 50,
      "messageSize": 256,
      "numberProducers": 2,
      "numberConsumers": 3,
      "duration": "10m"
    }'
  ```

- `POST /simulate/stop`: Stop an ongoing simulation
  ```bash
  curl -X POST http://localhost:2112/simulate/stop
  ```

### Metrics

- `GET /metrics`: Prometheus metrics endpoint

## Monitoring

The system includes a comprehensive monitoring setup:

1. **Prometheus**: Collects and stores metrics at http://localhost:9090
2. **Grafana**: Visualizes metrics with pre-configured dashboards at http://localhost:3000 (login: admin/pubsub123)

The dashboard includes:
- Message rates (published and delivered)
- Queue size
- Consumer count
- Consumer latency percentiles
- Batch size distribution
- Message size distribution
- Error rates

## Performance Tuning

### Benchmark Configuration

For optimal performance, adjust benchmark parameters:

- **MessageCount**: Number of messages to send
- **ConcurrentProducers**: Number of parallel producers
- **ConcurrentConsumers**: Number of parallel consumers
- **MessageSize**: Size of each message in bytes
- **WarmupCount**: Number of messages to send before measuring
- **RunDuration**: Maximum time the benchmark should run
- **PushInterval**: Time between automatic message pushes (0 disables auto-push)

The PushInterval configuration allows you to control how messages are delivered:
- Set to 0 (default): Messages are only pushed when explicitly requested
- Set to a duration (e.g., 100ms): Messages are automatically pushed at the specified interval

### Simulation Configuration

The continuous simulation can be configured with:

- **MessageRate**: Messages per second (default: 10)
- **MessageSize**: Size of each message in bytes (default: 1024)
- **NumberProducers**: Number of concurrent producers (default: 1)
- **NumberConsumers**: Number of concurrent consumers (default: 2)
- **Duration**: How long to run (e.g., "1h")
- **ConsumerLag**: Artificial lag to simulate network delays (e.g., "100ms")
- **PushInterval**: Time between automatic message pushes (default: 0, auto-push disabled)

The PushInterval setting determines how messages are delivered:
- When set to 0, messages stay in the queue until explicitly pushed
- When set to a duration, messages are automatically pushed to consumers at the specified interval
- Use smaller intervals (e.g., 100ms) for near real-time processing
- Use longer intervals (e.g., 1s) to batch more messages together

The message rate is automatically distributed across the number of producers. For example, if MessageRate is 100 and NumberProducers is 4, each producer will send 25 messages per second to maintain the total desired rate.

## Docker Compose Environment

The Docker Compose setup includes:

1. **PubSub Service**: The main application
2. **Prometheus**: For metrics collection
3. **Grafana**: For metrics visualization

Each service is configured and networked to work together seamlessly.

## Development

### Project Structure

- **model/**: Data structures for messages
- **pubsub/**: Core functionality
  - **pubsub.go**: Main service implementation
  - **batch_consumer.go**: Batch processing logic
  - **benchmark.go**: Performance testing
  - **metrics.go**: Prometheus metrics
  - **report.go**: System reporting

### Basic Usage Examples

```go
// Create a new PubSub instance with default settings (auto-push disabled)
ps := pubsub.NewPubSub()

// Create a consumer function
myConsumer := func(msg model.Message) {
    fmt.Printf("Received: %v\n", msg.Content)
}

// Subscribe it to the PubSub system
ps = pubsub.Subscribe(ps, myConsumer)

// Publish some messages
ps.Publish("Hello, World!")
ps.Publish("Another message")

// Push messages manually
ps.PushAll()  // Since auto-push is disabled by default
```

### Setting Up Auto-Push

```go
// Create PubSub with auto-push every 100ms
ps := pubsub.NewPubSub(pubsub.WithPushInterval(100 * time.Millisecond))

// Or update push interval later
ps = pubsub.SetPushInterval(ps, 500 * time.Millisecond)

// Start auto-push (messages will be pushed every 500ms)
ps = ps.StartAutoPush()

// Stop auto-push when done
ps = ps.StopAutoPush()
```

### Adding a Batch Consumer

```go
batchConfig := pubsub.NewBatchConsumerConfig().
    WithBatchSize(10).
    WithTimeout(5 * time.Second)

batchProcessor := func(batch []model.Message) {
    fmt.Printf("Processing batch of %d messages\n", len(batch))
}

ps = pubsub.SubscribeBatchConsumer(ps, batchConfig, batchProcessor)
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Inspired by functional programming principles
- Built with modern Go practices
- Monitoring powered by Prometheus and Grafana
