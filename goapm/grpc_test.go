package goapm

import (
	"context"
	"testing"
	"time"

	protos "github.com/hedon954/goapm/fixtures"
	"github.com/stretchr/testify/assert"
)

type helloSvc struct {
	protos.UnimplementedHelloServiceServer
}

func (s *helloSvc) SayHello(ctx context.Context, in *protos.HelloRequest) (*protos.HelloResponse, error) {
	return &protos.HelloResponse{Message: "Hello, " + in.Name}, nil
}

func TestGrpcServerAndClient_ShouldWork(t *testing.T) {
	server := NewGrpcServer(":50051")
	protos.RegisterHelloServiceServer(server, &helloSvc{})
	server.Start()

	time.Sleep(100 * time.Millisecond)

	client, err := NewGrpcClient("localhost:50051", "test server")
	assert.Nil(t, err)
	res, err := protos.NewHelloServiceClient(client).SayHello(context.Background(),
		&protos.HelloRequest{Name: "World"})
	assert.Nil(t, err)
	assert.Equal(t, "Hello, World", res.Message)
}
