# PubSub System with API

This is a functional PubSub system implementation with Prometheus metrics and a HTTP API.

## Running the System

```bash
# Build and run locally
go run main.go

# Or use Docker Compose
./run.sh
```

## API Endpoints

The system exposes several API endpoints on port 2112:

### GET /metrics

Prometheus metrics endpoint that exposes all the system metrics.

### POST /benchmark

Run a benchmark with configurable parameters.

**Parameters:**

Can be sent as JSON body or form parameters:

- `messageCount`: Number of messages to send
- `concurrentProducers`: Number of concurrent producers
- `concurrentConsumers`: Number of concurrent consumers
- `messageSize`: Size of each message in bytes
- `warmupCount`: Number of warmup messages
- `runDuration`: Maximum duration as string (e.g. "10s", "1m")

**Example:**

```bash
curl -X POST http://localhost:2112/benchmark \
  -H "Content-Type: application/json" \
  -d '{"messageCount": 1000, "concurrentProducers": 4, "messageSize": 512}'
```

### POST /simulate/start

Start a continuous message simulation.

**Parameters:**

Can be sent as JSON body or form parameters:

- `messageRate`: Messages per second (default: 10)
- `messageSize`: Size of each message in bytes (default: 1024)
- `duration`: Optional duration to run the simulation (e.g. "1h")
- `consumerLag`: Optional artificial lag for consumers (e.g. "100ms")

**Example:**

```bash
curl -X POST http://localhost:2112/simulate/start \
  -H "Content-Type: application/json" \
  -d '{"messageRate": 50, "messageSize": 256, "duration": "10m"}'
```

### POST /simulate/stop

Stop an ongoing simulation.

**Example:**

```bash
curl -X POST http://localhost:2112/simulate/stop
```

### GET /status

Get the current system status.

**Example:**

```bash
curl http://localhost:2112/status
```

## Metrics and Monitoring

The system includes a Grafana dashboard for monitoring metrics:

- Access Grafana at http://localhost:3000 (login: admin/pubsub123)
- The PubSub dashboard shows message rates, consumer latency, queue size, etc.

## Docker Compose

The included Docker Compose setup runs:

1. The PubSub application
2. Prometheus for metrics collection
3. Grafana for metrics visualization

For more details on the metrics, see the Grafana dashboard.
