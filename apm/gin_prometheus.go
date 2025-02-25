package apm

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// GinWithPrometheus wraps the gin router with prometheus metrics.
func GinWithPrometheus(r gin.IRouter, path string, middlewares ...gin.HandlerFunc) {
	handler := gin.WrapH(
		promhttp.HandlerFor(MetricsReg, promhttp.HandlerOpts{
			Registry: MetricsReg,
		}),
	)
	handlers := make([]gin.HandlerFunc, 0, len(middlewares)+1)
	handlers = append(handlers, middlewares...)
	handlers = append(handlers, handler)
	r.GET(path, handlers...)
}

// PrometheusBasicAuth is a middleware that authenticates requests with basic auth.
func PrometheusBasicAuth(username, password string) gin.HandlerFunc {
	return func(c *gin.Context) {
		un, pwd, ok := c.Request.BasicAuth()
		if !ok {
			c.Header("WWW-Authenticate", "Basic realm=Restricted")
			c.Status(http.StatusUnauthorized)
			c.Abort()
			return
		}
		if un != username || pwd != password {
			c.Header("WWW-Authenticate", "Basic realm=Restricted")
			c.Status(http.StatusUnauthorized)
			c.Abort()
			return
		}
	}
}
