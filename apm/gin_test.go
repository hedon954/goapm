package apm

import (
	"io"
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

	go func() {
		if err := router.Run(":12345"); err != nil {
			panic(err)
		}
	}()

	for {
		time.Sleep(10 * time.Millisecond)
		resp, err := http.Get("http://127.0.0.1:12345/")
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
