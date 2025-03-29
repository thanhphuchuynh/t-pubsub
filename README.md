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
    -d '{"messageRate": 50, "messageSize": 256, "duration": "10m"}'
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

### Simulation Configuration

The continuous simulation can be configured with:

- **MessageRate**: Messages per second (default: 10)
- **MessageSize**: Size of each message in bytes (default: 1024)
- **Duration**: How long to run (e.g., "1h")
- **ConsumerLag**: Artificial lag to simulate network delays (e.g., "100ms")

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

### Adding a Custom Consumer

```go
// Create a consumer function
myConsumer := func(msg model.Message) {
    fmt.Printf("Received: %v\n", msg.Content)
}

// Subscribe it to the PubSub system
ps = pubsub.Subscribe(ps, myConsumer)
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
