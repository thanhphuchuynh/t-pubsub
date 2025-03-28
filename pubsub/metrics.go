package pubsub

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricGetter defines an interface for getting metric values
type MetricGetter interface {
	GetValue() float64
}

// PrometheusCounter wraps prometheus.Counter with GetValue method
type PrometheusCounter struct {
	prometheus.Counter
}

func (pc PrometheusCounter) GetValue() float64 {
	// This is an approximation as prometheus doesn't expose direct value access
	m := make(chan prometheus.Metric, 1)
	pc.Collect(m)
	close(m)
	// In a real implementation we would parse the value, but for now return 0
	return 0
}

// PrometheusGauge wraps prometheus.Gauge with GetValue method
type PrometheusGauge struct {
	prometheus.Gauge
}

func (pg PrometheusGauge) GetValue() float64 {
	// This is an approximation as prometheus doesn't expose direct value access
	m := make(chan prometheus.Metric, 1)
	pg.Collect(m)
	close(m)
	// In a real implementation we would parse the value, but for now return 0
	return 0
}

// MetricsRegistry holds all the Prometheus metrics for the PubSub system
type MetricsRegistry struct {
	// Message metrics
	MessagesPublished PrometheusCounter
	MessagesDelivered PrometheusCounter
	MessageSize       prometheus.Histogram

	// Consumer metrics
	ConsumerCount   PrometheusGauge
	ConsumerLatency prometheus.Histogram

	// Queue metrics
	QueueSize PrometheusGauge
	BatchSize prometheus.Histogram

	// Error metrics
	ErrorsTotal PrometheusCounter
}

// Create a registry for our metrics
var (
	defaultRegistry = prometheus.NewRegistry()
	registryOnce    sync.Once
)

// GetRegistry returns the default Prometheus registry for our metrics
func GetRegistry() *prometheus.Registry {
	return defaultRegistry
}

// NewMetricsRegistry creates and registers Prometheus metrics
func NewMetricsRegistry(namespace string) *MetricsRegistry {
	if namespace == "" {
		namespace = "pubsub"
	}

	// Ensure we register with both the default registry and our custom one
	registry := &MetricsRegistry{
		// Message metrics
		MessagesPublished: PrometheusCounter{
			promauto.With(prometheus.DefaultRegisterer).NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "messages_published_total",
				Help:      "The total number of messages published",
			}),
		},
		MessagesDelivered: PrometheusCounter{
			promauto.With(prometheus.DefaultRegisterer).NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "messages_delivered_total",
				Help:      "The total number of messages delivered to consumers",
			}),
		},
		MessageSize: promauto.With(prometheus.DefaultRegisterer).NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "message_size_bytes",
			Help:      "Distribution of message sizes in bytes",
			Buckets:   prometheus.ExponentialBuckets(64, 2, 10),
		}),

		// Consumer metrics
		ConsumerCount: PrometheusGauge{
			promauto.With(prometheus.DefaultRegisterer).NewGauge(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "consumer_count",
				Help:      "The current number of consumers",
			}),
		},
		ConsumerLatency: promauto.With(prometheus.DefaultRegisterer).NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "consumer_latency_seconds",
			Help:      "Distribution of message processing latency in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10),
		}),

		// Queue metrics
		QueueSize: PrometheusGauge{
			promauto.With(prometheus.DefaultRegisterer).NewGauge(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "queue_size",
				Help:      "The current number of messages in the queue",
			}),
		},
		BatchSize: promauto.With(prometheus.DefaultRegisterer).NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "batch_size",
			Help:      "Distribution of batch sizes",
			Buckets:   prometheus.LinearBuckets(1, 1, 10),
		}),

		// Error metrics
		ErrorsTotal: PrometheusCounter{
			promauto.With(prometheus.DefaultRegisterer).NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "errors_total",
				Help:      "The total number of errors encountered",
			}),
		},
	}

	// Initialize default metrics values to ensure they show up even with zero values
	registry.ConsumerCount.Set(0)
	registry.QueueSize.Set(0)

	return registry
}

// Global default metrics registry
var defaultMetrics *MetricsRegistry

// Initialize the metrics registry once at package init time
func init() {
	defaultMetrics = NewMetricsRegistry("")
}

// GetDefaultMetrics returns the default metrics registry
func GetDefaultMetrics() *MetricsRegistry {
	return defaultMetrics
}
