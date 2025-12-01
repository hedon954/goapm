package apm

//nolint:all

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitMetricRegistry tests the initialization of the metric registry
func TestInitMetricRegistry(t *testing.T) {
	// Initialize the metric registry
	registry := InitMetricRegistry()

	// Verify registry is not nil
	assert.NotNil(t, registry)

	// Increment metrics
	ServerHandleCounter.WithLabelValues("type", "method", "peer", "peer_host").Inc()
	ServerHandleHistogram.WithLabelValues("type", "method", "status", "peer", "peer_host").Observe(0.2)
	ClientHandleCounter.WithLabelValues("type", "method", "server").Inc()
	ClientHandleHistogram.WithLabelValues("type", "method", "server").Observe(0.1)
	LibraryCounter.WithLabelValues("type", "method", "name", "server").Inc()

	// Verify all metrics are registered
	metrics, err := registry.Gather()
	require.NoError(t, err)
	foundMetrics := make(map[string]bool)
	for _, metric := range metrics {
		foundMetrics[*metric.Name] = true
	}

	assert.True(t, foundMetrics["server_handle_seconds"])
	assert.True(t, foundMetrics["server_handle_total"])
	assert.True(t, foundMetrics["client_handle_total"])
	assert.True(t, foundMetrics["client_handle_seconds"])
	assert.True(t, foundMetrics["lib_handle_total"])
}

// TestCustomLabels tests if custom labels are correctly added
func TestCustomLabels(t *testing.T) {
	// Create a registry with custom labels
	registry := newCustomMetricRegistry(map[string]string{
		"test_label1": "value1",
		"test_label2": "value2",
	})

	// Create and register a test metric
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "Counter for testing custom labels",
	})
	registry.MustRegister(counter)

	// Increment the counter
	counter.Inc()

	// Gather metrics
	families, err := registry.Gather()
	require.NoError(t, err)

	// Find our test metric
	var testFamily *io_prometheus_client.MetricFamily
	for _, family := range families {
		if *family.Name == "test_counter" {
			testFamily = family
			break
		}
	}

	// Verify metric was found
	require.NotNil(t, testFamily, "Test metric not found")
	require.Len(t, testFamily.Metric, 1, "Should have exactly one test metric")

	// Verify custom labels
	metric := testFamily.Metric[0]
	foundLabels := make(map[string]string)
	for _, label := range metric.Label {
		foundLabels[*label.Name] = *label.Value
	}

	assert.Equal(t, "value1", foundLabels["test_label1"])
	assert.Equal(t, "value2", foundLabels["test_label2"])
}

// TestMetricsIncrement tests incrementing metrics
func TestMetricsIncrement(t *testing.T) {
	// Reset registry to avoid interference
	MetricsReg = newCustomMetricRegistry(nil)
	MetricsReg.MustRegister(ServerHandleCounter)

	// Increment counter
	ServerHandleCounter.WithLabelValues(MetricTypeHTTP, "GET.test", "", "").Inc()

	// Gather metrics
	families, err := MetricsReg.Gather()
	require.NoError(t, err)

	// Find server handle counter
	var serverHandleFamily *io_prometheus_client.MetricFamily
	for _, family := range families {
		if *family.Name == "server_handle_total" {
			serverHandleFamily = family
			break
		}
	}

	// Verify metric was found
	require.NotNil(t, serverHandleFamily, "Server handle counter metric not found")

	// Verify label values and count
	found := false
	for _, metric := range serverHandleFamily.Metric {
		labelMap := make(map[string]string)
		for _, label := range metric.Label {
			labelMap[*label.Name] = *label.Value
		}

		if labelMap["type"] == MetricTypeHTTP && labelMap["method"] == "GET.test" {
			found = true
			assert.Equal(t, float64(1), *metric.Counter.Value)
			break
		}
	}

	assert.True(t, found, "Metric with correct labels not found")
}

// TestMetricsHistogram tests histogram metrics
func TestMetricsHistogram(t *testing.T) {
	// Reset registry to avoid interference
	MetricsReg = newCustomMetricRegistry(nil)
	MetricsReg.MustRegister(ServerHandleHistogram)

	// Observe histogram
	ServerHandleHistogram.WithLabelValues(MetricTypeHTTP, "GET.test", "200", "", "").Observe(0.1)
	ServerHandleHistogram.WithLabelValues(MetricTypeHTTP, "GET.test", "200", "", "").Observe(0.2)

	// Gather metrics
	families, err := MetricsReg.Gather()
	require.NoError(t, err)

	// Find server handle histogram
	var histogramFamily *io_prometheus_client.MetricFamily
	for _, family := range families {
		if *family.Name == "server_handle_seconds" {
			histogramFamily = family
			break
		}
	}

	// Verify metric was found
	require.NotNil(t, histogramFamily, "Server handle histogram metric not found")

	// Verify histogram values
	found := false
	for _, metric := range histogramFamily.Metric {
		labelMap := make(map[string]string)
		for _, label := range metric.Label {
			labelMap[*label.Name] = *label.Value
		}

		if labelMap["type"] == MetricTypeHTTP && labelMap["method"] == "GET.test" && labelMap["status"] == "200" {
			found = true
			assert.Equal(t, uint64(2), *metric.Histogram.SampleCount)
			assert.InEpsilon(t, 0.3, *metric.Histogram.SampleSum, 0.001)
			break
		}
	}

	assert.True(t, found, "Histogram with correct labels not found")
}
