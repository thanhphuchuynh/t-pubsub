package pubsub

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	dto "github.com/prometheus/client_model/go"
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
	var metric dto.Metric
	err := pc.Write(&metric)
	if err != nil {
		return 0
	}
	if metric.Counter != nil {
		return metric.Counter.GetValue()
	}
	return 0
}

// PrometheusGauge wraps prometheus.Gauge with GetValue method
type PrometheusGauge struct {
	prometheus.Gauge
}

func (pg PrometheusGauge) GetValue() float64 {
	var metric dto.Metric
	err := pg.Write(&metric)
	if err != nil {
		return 0
	}
	if metric.Gauge != nil {
		return metric.Gauge.GetValue()
	}
	return 0
}

// MetricsRegistry holds all the Prometheus metrics for the PubSub system
type MetricsRegistry struct {
	// Message metrics
	MessagesPublished    PrometheusCounter
	MessagesDelivered    PrometheusCounter
	MessageSize          prometheus.Histogram
	MessagesInQueue      PrometheusGauge
	MessagesCompleted    PrometheusCounter
	MessagesDropped      PrometheusCounter
	MessagesFailedToSend PrometheusCounter

	// Rate metrics
	ProducerRate prometheus.Gauge
	ConsumerRate prometheus.Gauge

	// Timing metrics
	ProcessingTime prometheus.Histogram
	QueueTime      prometheus.Histogram

	// Consumer metrics
	ConsumerCount   PrometheusGauge
	ConsumerLatency prometheus.Histogram
	ConsumersActive PrometheusGauge

	// Queue metrics
	QueueSize        PrometheusGauge
	QueueCapacity    PrometheusGauge
	BatchSize        prometheus.Histogram
	QueueUtilization prometheus.Gauge

	// Producer metrics
	ProducerCount   PrometheusGauge
	ProducersActive PrometheusGauge

	// Error metrics
	ErrorsTotal      PrometheusCounter
	ValidationErrors PrometheusCounter
	ProcessingErrors PrometheusCounter
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
		MessagesInQueue: PrometheusGauge{
			promauto.With(prometheus.DefaultRegisterer).NewGauge(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "messages_in_queue",
				Help:      "The current number of messages waiting in the queue",
			}),
		},
		MessagesCompleted: PrometheusCounter{
			promauto.With(prometheus.DefaultRegisterer).NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "messages_completed_total",
				Help:      "The total number of messages successfully processed",
			}),
		},
		MessagesDropped: PrometheusCounter{
			promauto.With(prometheus.DefaultRegisterer).NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "messages_dropped_total",
				Help:      "The total number of messages dropped due to queue full or other issues",
			}),
		},
		MessagesFailedToSend: PrometheusCounter{
			promauto.With(prometheus.DefaultRegisterer).NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "messages_failed_to_send_total",
				Help:      "The total number of messages that failed to be sent",
			}),
		},
		MessageSize: promauto.With(prometheus.DefaultRegisterer).NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "message_size_bytes",
			Help:      "Distribution of message sizes in bytes",
			Buckets:   prometheus.ExponentialBuckets(64, 2, 10),
		}),

		// Rate metrics
		ProducerRate: promauto.With(prometheus.DefaultRegisterer).NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "producer_message_rate",
			Help:      "Current rate of messages being produced per second",
		}),
		ConsumerRate: promauto.With(prometheus.DefaultRegisterer).NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "consumer_message_rate",
			Help:      "Current rate of messages being consumed per second",
		}),

		// Timing metrics
		ProcessingTime: promauto.With(prometheus.DefaultRegisterer).NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "message_processing_duration_seconds",
			Help:      "Time taken to process messages",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10),
		}),
		QueueTime: promauto.With(prometheus.DefaultRegisterer).NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "message_queue_duration_seconds",
			Help:      "Time messages spend in the queue before being processed",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10),
		}),

		// Consumer metrics
		ConsumerCount: PrometheusGauge{
			promauto.With(prometheus.DefaultRegisterer).NewGauge(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "consumer_count",
				Help:      "The total number of registered consumers",
			}),
		},
		ConsumersActive: PrometheusGauge{
			promauto.With(prometheus.DefaultRegisterer).NewGauge(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "consumers_active",
				Help:      "The number of currently active consumers",
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
		QueueCapacity: PrometheusGauge{
			promauto.With(prometheus.DefaultRegisterer).NewGauge(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "queue_capacity",
				Help:      "The maximum capacity of the queue",
			}),
		},
		QueueUtilization: promauto.With(prometheus.DefaultRegisterer).NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "queue_utilization_percent",
			Help:      "The current queue utilization as a percentage",
		}),
		BatchSize: promauto.With(prometheus.DefaultRegisterer).NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "batch_size",
			Help:      "Distribution of batch sizes",
			Buckets:   prometheus.LinearBuckets(1, 1, 10),
		}),

		// Producer metrics
		ProducerCount: PrometheusGauge{
			promauto.With(prometheus.DefaultRegisterer).NewGauge(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "producer_count",
				Help:      "The total number of registered producers",
			}),
		},
		ProducersActive: PrometheusGauge{
			promauto.With(prometheus.DefaultRegisterer).NewGauge(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "producers_active",
				Help:      "The number of currently active producers",
			}),
		},

		// Error metrics
		ErrorsTotal: PrometheusCounter{
			promauto.With(prometheus.DefaultRegisterer).NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "errors_total",
				Help:      "The total number of errors encountered",
			}),
		},
		ValidationErrors: PrometheusCounter{
			promauto.With(prometheus.DefaultRegisterer).NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "validation_errors_total",
				Help:      "The total number of message validation errors",
			}),
		},
		ProcessingErrors: PrometheusCounter{
			promauto.With(prometheus.DefaultRegisterer).NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "processing_errors_total",
				Help:      "The total number of message processing errors",
			}),
		},
	}

	// Initialize default metrics values
	registry.ConsumerCount.Set(0)
	registry.ConsumersActive.Set(0)
	registry.ProducerCount.Set(0)
	registry.ProducersActive.Set(0)
	registry.QueueSize.Set(0)
	registry.QueueCapacity.Set(1000) // Default capacity
	registry.QueueUtilization.Set(0)
	registry.ProducerRate.Set(0)
	registry.ConsumerRate.Set(0)

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
