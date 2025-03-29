package pubsub

import (
	"testing"
)

func TestMetricsRegistry(t *testing.T) {
	metrics := NewMetricsRegistry("test")

	t.Run("new metrics registry", func(t *testing.T) {
		if metrics == nil {
			t.Error("Expected non-nil metrics registry")
		}

		// Test counter initialization
		if metrics.MessagesPublished.GetValue() != 0 {
			t.Error("Expected MessagesPublished to start at 0")
		}
		if metrics.MessagesDelivered.GetValue() != 0 {
			t.Error("Expected MessagesDelivered to start at 0")
		}

		// Test gauge initialization
		queueCapValue := metrics.QueueCapacity.GetValue()
		if queueCapValue != 1000 {
			t.Errorf("Expected QueueCapacity to be 1000, got %f", queueCapValue)
		}

		if metrics.QueueSize.GetValue() != 0 {
			t.Error("Expected QueueSize to start at 0")
		}
	})

	t.Run("counter operations", func(t *testing.T) {
		initial := metrics.MessagesPublished.GetValue()

		// Test increment
		metrics.MessagesPublished.Inc()
		after := metrics.MessagesPublished.GetValue()
		if after != initial+1 {
			t.Errorf("Expected counter to increment by 1, got %f", after-initial)
		}
	})

	t.Run("gauge operations", func(t *testing.T) {
		testValue := float64(42)

		// Test set
		metrics.QueueSize.Set(testValue)
		if metrics.QueueSize.GetValue() != testValue {
			t.Errorf("Expected gauge value %f, got %f", testValue, metrics.QueueSize.GetValue())
		}

		// Test increment/decrement
		metrics.QueueSize.Inc()
		if metrics.QueueSize.GetValue() != testValue+1 {
			t.Error("Gauge increment failed")
		}

		metrics.QueueSize.Dec()
		if metrics.QueueSize.GetValue() != testValue {
			t.Error("Gauge decrement failed")
		}
	})

	t.Run("default metrics singleton", func(t *testing.T) {
		metrics1 := GetDefaultMetrics()
		metrics2 := GetDefaultMetrics()

		if metrics1 != metrics2 {
			t.Error("GetDefaultMetrics should return the same instance")
		}
	})

	t.Run("custom namespace", func(t *testing.T) {
		customMetrics := NewMetricsRegistry("custom")

		// Both should be valid and initialized
		if customMetrics == nil {
			t.Error("Failed to create metrics with different namespaces")
		}
	})

	t.Run("prometheus getter interface", func(t *testing.T) {
		// Test Counter implements MetricGetter
		var _ MetricGetter = metrics.MessagesPublished

		// Test Gauge implements MetricGetter
		var _ MetricGetter = metrics.QueueSize
	})
}

func TestMetricOperations(t *testing.T) {
	metrics := GetDefaultMetrics()

	t.Run("message metrics", func(t *testing.T) {
		initial := metrics.MessagesPublished.GetValue()
		metrics.MessagesPublished.Inc()
		if metrics.MessagesPublished.GetValue() != initial+1 {
			t.Error("Failed to increment message count")
		}
	})

	t.Run("queue metrics", func(t *testing.T) {
		// Test queue size updates
		metrics.QueueSize.Set(0)
		metrics.QueueSize.Inc()
		if metrics.QueueSize.GetValue() != 1 {
			t.Error("Failed to increment queue size")
		}

		// Test queue utilization
		metrics.QueueCapacity.Set(100)
		metrics.QueueSize.Set(50)
		metrics.QueueUtilization.Set(50.0) // 50% utilization

		// For unwrapped prometheus.Gauge, we can only verify that Set() doesn't panic
		// as we can't directly read the value without prometheus test helpers
	})

	t.Run("consumer metrics", func(t *testing.T) {
		initial := metrics.ConsumerCount.GetValue()
		metrics.ConsumerCount.Inc()
		if metrics.ConsumerCount.GetValue() != initial+1 {
			t.Error("Failed to increment consumer count")
		}
	})

	t.Run("error metrics", func(t *testing.T) {
		initial := metrics.ErrorsTotal.GetValue()
		metrics.ErrorsTotal.Inc()
		if metrics.ErrorsTotal.GetValue() != initial+1 {
			t.Error("Failed to increment error count")
		}
	})
}
