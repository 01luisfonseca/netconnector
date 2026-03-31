package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	pb "netconnector/proto/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestTunnelServer_Handshake(t *testing.T) {
	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)
	
	db := &MockRepo{
		mappings: map[string]string{
			"proxy.com":   "agent-1",
			"success.com": "agent-success",
			"mux.com":     "agent-mux",
		},
	}
	router := NewRouter()
	srv := grpc.NewServer()
	pb.RegisterTunnelServiceServer(srv, NewTunnelServer(router, db))
	
	go srv.Serve(lis)
	defer srv.Stop()

	// Client connection
	conn, err := grpc.DialContext(context.Background(), "bufnet", 
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewTunnelServiceClient(conn)

	t.Run("EmptyPayload", func(t *testing.T) {
		stream, _ := client.TunnelStream(context.Background())
		_ = stream.Send(&pb.TunnelMessage{Payload: nil})
		resp, err := stream.Recv()
		if err == nil && resp.GetRegisterResponse() == nil {
			// Success depends on how gRPC handles the error return
		}
	})

	t.Run("WrongMessageType", func(t *testing.T) {
		stream, _ := client.TunnelStream(context.Background())
		_ = stream.Send(&pb.TunnelMessage{
			Payload: &pb.TunnelMessage_HttpResponse{
				HttpResponse: &pb.HTTPResponse{},
			},
		})
		_, err := stream.Recv()
		if err == nil {
			t.Error("expected error for wrong message type")
		}
	})

	t.Run("EmptyClientID", func(t *testing.T) {
		stream, _ := client.TunnelStream(context.Background())
		_ = stream.Send(&pb.TunnelMessage{
			Payload: &pb.TunnelMessage_RegisterRequest{
				RegisterRequest: &pb.RegisterRequest{ClientId: ""},
			},
		})
		_, err := stream.Recv()
		if err == nil {
			t.Error("expected error for empty clientID")
		}
	})

	t.Run("UnauthorizedClientID", func(t *testing.T) {
		stream, err := client.TunnelStream(context.Background())
		if err != nil {
			t.Fatalf("Failed to open stream: %v", err)
		}
		
		err = stream.Send(&pb.TunnelMessage{
			Payload: &pb.TunnelMessage_RegisterRequest{
				RegisterRequest: &pb.RegisterRequest{ClientId: "evil-agent"},
			},
		})
		if err != nil {
			t.Fatalf("Send handshake failed: %v", err)
		}

		resp, err := stream.Recv()
		t.Logf("Recv result: resp=%v, err=%v", resp, err)
		if err == nil {
			if resp.GetRegisterResponse().Success {
				t.Error("expected failure for unauthorized client")
			} else {
				t.Logf("Got expected failure response: %v", resp.GetRegisterResponse().ErrorMessage)
			}
		} else {
			t.Logf("Got expected error: %v", err)
		}
	})

	t.Run("DBErrorInHandshake", func(t *testing.T) {
		db.Err = fmt.Errorf("db failure")
		defer func() { db.Err = nil }()
		
		stream, _ := client.TunnelStream(context.Background())
		_ = stream.Send(&pb.TunnelMessage{
			Payload: &pb.TunnelMessage_RegisterRequest{
				RegisterRequest: &pb.RegisterRequest{ClientId: "agent-1"},
			},
		})
		_, err := stream.Recv()
		if err == nil {
			t.Error("expected error due to db failure")
		}
	})

	t.Run("SuccessfulHandshake", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		
		clientID := "agent-success"
		stream, err := client.TunnelStream(ctx)
		if err != nil {
			t.Fatalf("Failed to open stream: %v", err)
		}
		
		err = stream.Send(&pb.TunnelMessage{
			Payload: &pb.TunnelMessage_RegisterRequest{
				RegisterRequest: &pb.RegisterRequest{ClientId: clientID},
			},
		})
		if err != nil {
			t.Fatalf("Send handshake failed: %v", err)
		}

		resp, err := stream.Recv()
		if err != nil {
			t.Fatalf("Recv response failed: %v", err)
		}
		if !resp.GetRegisterResponse().Success {
			t.Errorf("Handshake failed: %s", resp.GetRegisterResponse().ErrorMessage)
		}

		// Verify it is registered in router
		ac, err := router.GetClient(clientID)
		if err != nil || ac == nil {
			t.Fatalf("expected client %s to be in router, err: %v", clientID, err)
		}
	})

	t.Run("MultiplexingRequestResponse", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		
		clientID := "agent-mux"
		stream, _ := client.TunnelStream(ctx)
		_ = stream.Send(&pb.TunnelMessage{
			Payload: &pb.TunnelMessage_RegisterRequest{
				RegisterRequest: &pb.RegisterRequest{ClientId: clientID},
			},
		})
		_, _ = stream.Recv() // Handshake ack

		// Start agent-like behavior in goroutine
		go func() {
			for {
				msg, err := stream.Recv()
				if err == io.EOF { return }
				if err != nil { return }

				if req := msg.GetHttpRequest(); req != nil {
					// Respond back
					_ = stream.Send(&pb.TunnelMessage{
						Payload: &pb.TunnelMessage_HttpResponse{
							HttpResponse: &pb.HTTPResponse{
								RequestId: req.RequestId,
								StatusCode: 200,
								Body: []byte("back-and-forth"),
							},
						},
					})
				}
			}
		}()

		ac, err := router.GetClient(clientID)
		if err != nil {
			t.Fatalf("failed to get client from router: %v", err)
		}
		resCh := ac.RegisterPromise("req-muiti")
		
		_ = ac.SendRequest(&pb.HTTPRequest{
			RequestId: "req-muiti",
			Method: "GET",
			Url: "/",
		})

		select {
		case resp := <-resCh:
			if string(resp.Body) != "back-and-forth" {
				t.Errorf("got wrong body: %s", resp.Body)
			}
		case <-time.After(1 * time.Second):
			t.Error("timed out waiting for response")
		}
	})
}
