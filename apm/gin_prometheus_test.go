package apm

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGinWithPrometheus(t *testing.T) {
	router := gin.New()
	GinWithPrometheus(router, "/goapm/metrics", PrometheusBasicAuth("admin", "123456"))

	// test unauthenticated
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/goapm/metrics", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 401, w.Code)

	// test authenticated
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/goapm/metrics", nil)
	req.SetBasicAuth("admin", "123456")
	router.ServeHTTP(w, req)
	assert.NotEmpty(t, w.Body.String())
}
