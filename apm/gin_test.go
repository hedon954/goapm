package apm

import (
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard
}

func TestGinServer_Handle(t *testing.T) {
	router := gin.Default()
	router.Use(GinOtel())
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, World!")
	})

	listener, err := net.Listen("tcp", ":") //nolint:gosec
	if err != nil {
		panic(err)
	}

	go func() {
		if err := router.RunListener(listener); err != nil {
			panic(err)
		}
	}()

	for {
		time.Sleep(10 * time.Millisecond)
		resp, err := http.Get("http://" + listener.Addr().String() + "/")
		if err != nil {
			continue
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("Failed to close response body: %v", err)
		}
		if resp.StatusCode == http.StatusOK {
			break
		}
	}
}
