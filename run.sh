#!/bin/bash

echo "Setting up directories..."
mkdir -p prometheus
mkdir -p grafana/provisioning/datasources
mkdir -p grafana/dashboards

echo "Starting Docker Compose..."
docker compose up -d

echo "Waiting for services to start..."
sleep 15  # Increased wait time for services to properly initialize

echo "Checking if services are running..."
docker compose ps

echo "Checking if PubSub health endpoint is accessible from inside container:"
docker compose exec pubsub wget -q -O- http://localhost:2112/health || echo "Health endpoint not responding from container"

echo "Checking if PubSub metrics are accessible from inside container:"
docker compose exec pubsub wget -q -O- http://localhost:2112/metrics | head -n 10 || echo "Metrics endpoint not responding from container"

echo "Checking if PubSub health endpoint is accessible from local machine:"
curl -s http://localhost:2112/health || wget -q -O- http://localhost:2112/health || echo "Health endpoint not responding from local machine"

echo "Testing benchmark API from inside container:"
docker compose exec pubsub wget -q -O- --header="Content-Type: application/json" \
  --post-data='{"messageCount": 10, "concurrentProducers": 1, "messageSize": 128}' \
  http://localhost:2112/benchmark || echo "Benchmark API not responding from container"

echo "PubSub Demo is now running!"
echo "Access the services at:"
echo "- Grafana: http://localhost:3000 (admin/pubsub123)"
echo "- Prometheus: http://localhost:9090"
echo "- PubSub Metrics: http://localhost:2112/metrics"
echo "- PubSub Health: http://localhost:2112/health"
echo "- PubSub API: http://localhost:2112/status"

echo "To stop the demo, run: docker compose down"
