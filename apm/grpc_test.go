package apm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	protos "github.com/hedon954/goapm/fixtures"
)

type helloSvc struct {
	protos.UnimplementedHelloServiceServer
}

func (s *helloSvc) SayHello(ctx context.Context, in *protos.HelloRequest) (*protos.HelloResponse, error) {
	return &protos.HelloResponse{Message: "Hello, " + in.Name}, nil
}

func TestGrpcServerAndClient_ShouldWork(t *testing.T) {
	server := NewGrpcServer(":")
	protos.RegisterHelloServiceServer(server, &helloSvc{})
	server.Start()
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	client, err := NewGrpcClient(server.listener.Addr().String(), "test server")
	assert.Nil(t, err)
	res, err := protos.NewHelloServiceClient(client).SayHello(context.Background(),
		&protos.HelloRequest{Name: "World"})
	assert.Nil(t, err)
	assert.Equal(t, "Hello, World", res.Message)
}
