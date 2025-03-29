FROM golang:1.18-alpine as builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod ./

# Download Go modules
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o pubsub-app .

# Create a minimal runtime image
FROM alpine:latest

WORKDIR /app

# Install only essential tools - just wget for healthchecks and diagnostics
RUN apk --no-cache add ca-certificates wget

# Copy the binary from the builder stage
COPY --from=builder /app/pubsub-app ./pubsub-app

# Expose the metrics port
EXPOSE 2112

# Add healthcheck using wget
HEALTHCHECK --interval=5s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O- http://localhost:2112/health || exit 1

# Run the application
CMD ["./pubsub-app"]
