package main

import (
	"github.com/hedon954/go-apm/examples/ordersvc/metric"
	"github.com/hedon954/goapm"
	"github.com/hedon954/goapm/apm"
)

func main() {
	// init infra
	goapm.NewInfra("ordersvc",
		goapm.WithMySQL("root:root@tcp(apm-mysql:3306)/ordersvc?charset=utf8mb4&parseTime=True&loc=Local", "ordersvc"),
		goapm.WithAPM("goapm-otel-collector:4317"),
		goapm.WithMetrics(metric.All()...),
		goapm.WithAutoPProf(&apm.AutoPProfOpt{
			EnableCPU:       true,
			EnableMem:       true,
			EnableGoroutine: true,
		}),
	)

	// init grpc clients

}
