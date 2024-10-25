package goapm

import (
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	io_prometheus_client "github.com/prometheus/client_model/go"

	"github.com/hedon954/goapm/goapm/internal"
)

const (
	MetricTypeHTTP = "http"
	MetricTypeGRPC = "grpc"

	LibraryTypeMySQL = "mysql"
	LibraryTypeRedis = "redis"
)

func init() {
	MetricsReg.MustRegister(serverHandleHistogram, serverHandleCounter, clientHandleCounter, clientHandleHistogram, libraryCounter)
	MetricsReg.MustRegister(
		collectors.NewGoCollector(
			collectors.WithGoCollectorRuntimeMetrics(collectors.GoRuntimeMetricsRule{
				Matcher: regexp.MustCompile("/.*"),
			}),
		),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
}

var (
	// MetricsReg is the global metric registry.
	MetricsReg = newCustomMetricRegistry(map[string]string{
		"host": internal.BuildInfo.Hostname(),
		"app":  internal.BuildInfo.AppName(),
	})
)

var (
	serverHandleHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "server_handle_seconds",
		Help: "The duration of the server handle",
	}, []string{"type", "method", "status", "peer", "peer_host"})

	serverHandleCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "server_handle_total",
		Help: "The total number of server handle",
	}, []string{"type", "method", "peer", "peer_host"})

	clientHandleCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "client_handle_total",
		Help: "The total number of client handle",
	}, []string{"type", "method", "server"})

	clientHandleHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "client_handle_seconds",
		Help: "The duration of the client handle",
	}, []string{"type", "method", "server"})

	libraryCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lib_handle_total",
		Help: "The total number of third party library handle",
	}, []string{"type", "method", "name", "server"})
)

// customMetricRegistry is a wrapper of prometheus.Registry.
// it adds custom labels to the metrics
type customMetricRegistry struct {
	*prometheus.Registry
	customLabels []*io_prometheus_client.LabelPair
}

func newCustomMetricRegistry(labels map[string]string) *customMetricRegistry {
	c := &customMetricRegistry{
		Registry: prometheus.NewRegistry(),
	}

	for k, v := range labels {
		tmpK := k
		tmpV := v
		c.customLabels = append(c.customLabels, &io_prometheus_client.LabelPair{
			Name:  &tmpK,
			Value: &tmpV,
		})
	}

	return c
}

// Gather calls the Collect method of the registered Collectors and then
// gathers the collected metrics into a lexicographically sorted slice
// of uniquely named MetricFamily protobufs. Gather ensures that the
// returned slice is valid and self-consistent so that it can be used
// for valid exposition. As an exception to the strict consistency
// requirements described for metric.Desc, Gather will tolerate
// different sets of label names for metrics of the same metric family.
//
// Even if an error occurs, Gather attempts to gather as many metrics as
// possible. Hence, if a non-nil error is returned, the returned
// MetricFamily slice could be nil (in case of a fatal error that
// prevented any meaningful metric collection) or contain a number of
// MetricFamily protobufs, some of which might be incomplete, and some
// might be missing altogether. The returned error (which might be a
// MultiError) explains the details. Note that this is mostly useful for
// debugging purposes. If the gathered protobufs are to be used for
// exposition in actual monitoring, it is almost always better to not
// expose an incomplete result and instead disregard the returned
// MetricFamily protobufs in case the returned error is non-nil.
func (c *customMetricRegistry) Gather() ([]*io_prometheus_client.MetricFamily, error) {
	metricFamilies, err := c.Registry.Gather()
	for _, mf := range metricFamilies {
		metrics := mf.Metric
		for _, metric := range metrics {
			metric.Label = append(metric.Label, c.customLabels...)
		}
	}
	return metricFamilies, err
}
