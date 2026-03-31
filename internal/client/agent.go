package client

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"netconnector/pkg/logger"
	pb "netconnector/proto/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// Agent manages the connection to the VPS
type Agent struct {
	serverAddr string
	clientID   string
	executor   *Executor
	// Dialer for testing
	dialer func(ctx context.Context, addr string, opts []grpc.DialOption) (*grpc.ClientConn, error)
}

func NewAgent(serverAddr, clientID string, executor *Executor) *Agent {
	return &Agent{
		serverAddr: serverAddr,
		clientID:   clientID,
		executor:   executor,
		dialer: func(ctx context.Context, addr string, opts []grpc.DialOption) (*grpc.ClientConn, error) {
			return grpc.NewClient(addr, opts...)
		},
	}
}

func (a *Agent) Start(ctx context.Context) error {
	kacp := keepalive.ClientParameters{
		Time:                10 * time.Second, // send pings every 10 seconds if there is no activity
		Timeout:             time.Second,      // wait 1 second for ping ack before considering the connection dead
		PermitWithoutStream: true,             // send pings even without active streams
	}

	for {
		err := a.connectAndReceive(ctx, kacp)
		if err != nil {
			logger.Error("Connection dropped, retrying in 5 seconds...", "err", err)
		}

		select {
		case <-ctx.Done():
			logger.Info("Agent shutting down")
			return nil
		case <-time.After(5 * time.Second): // Exponential backoff could be implemented here
			continue
		}
	}
}

func (a *Agent) connectAndReceive(ctx context.Context, kacp keepalive.ClientParameters) error {
	logger.Info("Connecting to VPS", "address", a.serverAddr)

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithKeepaliveParams(kacp))

	insecureMode := os.Getenv("GRPC_INSECURE") == "true"
	if insecureMode {
		logger.Warn("🚨 gRPC TLS disabled via GRPC_INSECURE=true")
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		// Use TLS
		tlsCAFile := os.Getenv("TLS_CA_FILE")
		if tlsCAFile != "" {
			creds, err := credentials.NewClientTLSFromFile(tlsCAFile, "")
			if err != nil {
				return fmt.Errorf("failed to load TLS CA file %s: %w", tlsCAFile, err)
			}
			opts = append(opts, grpc.WithTransportCredentials(creds))
			logger.Info("gRPC TLS Enabled using custom CA", "ca_file", tlsCAFile)
		} else {
			// Rely on system certificates (e.g., Let's Encrypt)
			opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(nil)))
			logger.Info("gRPC TLS Enabled using system CA")
		}
	}

	conn, err := a.dialer(ctx, a.serverAddr, opts)
	if err != nil {
		return fmt.Errorf("failed to dial server: %w", err)
	}
	defer conn.Close()

	client := pb.NewTunnelServiceClient(conn)
	stream, err := client.TunnelStream(ctx)
	if err != nil {
		return fmt.Errorf("failed to open stream: %w", err)
	}

	// 1. Send Handshake
	err = stream.Send(&pb.TunnelMessage{
		Payload: &pb.TunnelMessage_RegisterRequest{
			RegisterRequest: &pb.RegisterRequest{
				ClientId: a.clientID,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to send handshake: %w", err)
	}

	// 2. Wait for Handshake ACK
	msg, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive handshake ack: %w", err)
	}

	payload := msg.GetPayload()
	if payload == nil {
		return fmt.Errorf("empty payload in handshake ack")
	}

	ack, ok := payload.(*pb.TunnelMessage_RegisterResponse)
	if !ok {
		return fmt.Errorf("expected RegisterResponse, got something else")
	}

	if !ack.RegisterResponse.GetSuccess() {
		return fmt.Errorf("server rejected handshake: %s", ack.RegisterResponse.GetErrorMessage())
	}

	logger.Info("Successfully registered with VPS", "client_id", a.clientID)

	// 3. Enter Event Loop
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		inMsg, err := stream.Recv()
		if err == io.EOF {
			return fmt.Errorf("server closed stream")
		}
		if err != nil {
			return fmt.Errorf("stream recv error: %w", err)
		}

		resPayload, ok := inMsg.GetPayload().(*pb.TunnelMessage_HttpRequest)
		if !ok {
			logger.Warn("Received non-HTTP request message from server")
			continue
		}

		pbReq := resPayload.HttpRequest

		// 4. Handle HTTP execution in a separate goroutine
		// We pass the stream to the executor so it can write back.
		// Executor will handle the send mutex.
		go a.executor.Handle(pbReq, stream)
	}
}
