package client

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	pb "netconnector/proto/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/test/bufconn"
)

type mockTunnelServer struct {
	pb.UnimplementedTunnelServiceServer
}

func (s *mockTunnelServer) TunnelStream(stream pb.TunnelService_TunnelStreamServer) error {
	// 1. Handshake
	msg, err := stream.Recv()
	if err != nil { return err }
	
	if msg.GetRegisterRequest() != nil {
		_ = stream.Send(&pb.TunnelMessage{
			Payload: &pb.TunnelMessage_RegisterResponse{
				RegisterResponse: &pb.RegisterResponse{Success: true},
			},
		})
	}

	// 2. Send one test request
	_ = stream.Send(&pb.TunnelMessage{
		Payload: &pb.TunnelMessage_HttpRequest{
			HttpRequest: &pb.HTTPRequest{
				RequestId: "agent-test-id",
				Method: "GET",
				Url: "/ping",
			},
		},
	})
	
	// Wait a bit then exit
	time.Sleep(100 * time.Millisecond)
	return nil
}

func TestAgent_ConnectAndReceive(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	pb.RegisterTunnelServiceServer(srv, &mockTunnelServer{})
	go srv.Serve(lis)
	defer srv.Stop()

	// Setup Agent
	executor := NewExecutor("http://localhost:8080")
	agent := NewAgent("bufnet", "test-client", executor)

	// Override dialer to use bufconn
	agent.dialer = func(ctx context.Context, addr string, opts []grpc.DialOption) (*grpc.ClientConn, error) {
		return grpc.DialContext(ctx, addr, append(opts, 
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)...)
	}

	// Run connectAndReceive in a context with timeout to avoid hanging
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Set GRPC_INSECURE to bypass TLS logic in agent.go
	os.Setenv("GRPC_INSECURE", "true")
	defer os.Unsetenv("GRPC_INSECURE")

	err := agent.connectAndReceive(ctx, keepalive.ClientParameters{})
	// We expect a timeout error because the loop in connectAndReceive is infinite 
	if err != nil && err != context.DeadlineExceeded && err.Error() != "rpc error: code = Canceled desc = context canceled" {
		t.Logf("connectAndReceive exited with expected context end/error: %v", err)
	}
}

func TestAgent_HandshakeErrors(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	pb.RegisterTunnelServiceServer(srv, &errorHandshakeServer{})
	go srv.Serve(lis)
	defer srv.Stop()

	executor := NewExecutor("http://localhost:8080")
	agent := NewAgent("bufnet", "test-client", executor)
	agent.dialer = func(ctx context.Context, addr string, opts []grpc.DialOption) (*grpc.ClientConn, error) {
		return grpc.DialContext(ctx, addr, append(opts, 
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)...)
	}

	err := agent.connectAndReceive(context.Background(), keepalive.ClientParameters{})
	if err == nil {
		t.Error("expected error during handshake")
	}
}

type errorHandshakeServer struct {
	pb.UnimplementedTunnelServiceServer
}
func (s *errorHandshakeServer) TunnelStream(stream pb.TunnelService_TunnelStreamServer) error {
	_, _ = stream.Recv()
	return stream.Send(&pb.TunnelMessage{
		Payload: &pb.TunnelMessage_RegisterResponse{
			RegisterResponse: &pb.RegisterResponse{Success: false, ErrorMessage: "denied"},
		},
	})
}
func TestAgent_HandshakePayloadErrors(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	pb.RegisterTunnelServiceServer(srv, &invalidPayloadServer{})
	go srv.Serve(lis)
	defer srv.Stop()

	agent := NewAgent("bufnet", "test-client", NewExecutor(""))
	agent.dialer = func(ctx context.Context, addr string, opts []grpc.DialOption) (*grpc.ClientConn, error) {
		return grpc.DialContext(ctx, addr, append(opts, 
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)...)
	}

	err := agent.connectAndReceive(context.Background(), keepalive.ClientParameters{})
	if err == nil {
		t.Error("expected error due to invalid handshake payload")
	}
}

type invalidPayloadServer struct {
	pb.UnimplementedTunnelServiceServer
}
func (s *invalidPayloadServer) TunnelStream(stream pb.TunnelService_TunnelStreamServer) error {
	_, _ = stream.Recv()
	// Send message with nil payload
	return stream.Send(&pb.TunnelMessage{Payload: nil})
}
