package metric

import "github.com/prometheus/client_golang/prometheus"

var (
	OrderSuccessCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "order_success_total",
		Help: "The total number of orders that have been successfully processed",
	}, []string{"sku_id"})
)

func All() []prometheus.Collector {
	return []prometheus.Collector{
		OrderSuccessCounter,
	}
}
