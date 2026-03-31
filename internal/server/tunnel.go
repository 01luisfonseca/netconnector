package server

import (
	"io"
	"netconnector/pkg/logger"
	pb "netconnector/proto/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TunnelServer implementations pb.TunnelServiceServer
type TunnelServer struct {
	pb.UnimplementedTunnelServiceServer
	router *Router
	db     Repository
}

func NewTunnelServer(router *Router, db Repository) *TunnelServer {
	return &TunnelServer{
		router: router,
		db:     db,
	}
}

func (s *TunnelServer) TunnelStream(stream pb.TunnelService_TunnelStreamServer) error {
	logger.Info("New client stream connected, waiting for handshake...")

	// 1. Wait for handshake (RegisterRequest)
	in, err := stream.Recv()
	if err != nil {
		logger.Error("Failed to receive handshake", "err", err)
		return err
	}

	payload := in.GetPayload()
	if payload == nil {
		return status.Error(codes.InvalidArgument, "Payload empty or missing")
	}

	req, ok := payload.(*pb.TunnelMessage_RegisterRequest)
	if !ok {
		logger.Error("First message was not a register request")
		return status.Error(codes.FailedPrecondition, "Expected RegisterRequest as first message")
	}

	clientID := req.RegisterRequest.GetClientId()
	if clientID == "" {
		return status.Error(codes.InvalidArgument, "client_id cannot be empty")
	}

	// 2. Validate Authorization
	isRegistered, err := s.db.IsClientIDRegistered(clientID)
	if err != nil {
		logger.Error("Database error while verifying client_id", "client_id", clientID, "err", err)
		return status.Error(codes.Internal, "Internal server error during handshake")
	}
	if !isRegistered {
		logger.Warn("Rejected connection from unregistered client", "client_id", clientID)
		_ = stream.Send(&pb.TunnelMessage{
			Payload: &pb.TunnelMessage_RegisterResponse{
				RegisterResponse: &pb.RegisterResponse{Success: false, ErrorMessage: "unauthorized client_id"},
			},
		})
		return status.Error(codes.PermissionDenied, "unauthorized client_id")
	}

	// 3. Register inside our router memory
	client, err := s.router.RegisterClient(clientID, stream)
	if err != nil {
		_ = stream.Send(&pb.TunnelMessage{
			Payload: &pb.TunnelMessage_RegisterResponse{
				RegisterResponse: &pb.RegisterResponse{Success: false, ErrorMessage: err.Error()},
			},
		})
		return err
	}

	// Make sure to unregister when the stream dies
	defer s.router.UnregisterClient(clientID)

	// Acknowledge handshake
	err = stream.Send(&pb.TunnelMessage{
		Payload: &pb.TunnelMessage_RegisterResponse{
			RegisterResponse: &pb.RegisterResponse{Success: true},
		},
	})
	if err != nil {
		logger.Error("Failed to send handshake acknowledgment", "err", err)
		return err
	}

	logger.Info("Handshake complete, entering listener loop", "client_id", clientID)

	// 4. Listener Loop: Read incoming HTTP responses
	for {
		inMsg, err := stream.Recv()
		if err == io.EOF {
			logger.Info("Client disconnected gracefully", "client_id", clientID)
			return nil
		}
		if err != nil {
			// e.g. disconnect, context cancel, stream end
			logger.Warn("Stream error", "client_id", clientID, "err", err)
			return err
		}

		resPayload := inMsg.GetPayload()
		switch v := resPayload.(type) {
		case *pb.TunnelMessage_HttpResponse:
			// Match request_id to our pending channel
			reqID := v.HttpResponse.GetRequestId()
			client.ResolvePromise(reqID, v.HttpResponse)

		case *pb.TunnelMessage_RegisterRequest:
			// Invalid at this state
			logger.Warn("Received another handshake mid-stream", "client_id", clientID)

		default:
			logger.Warn("Unknown message type received from client", "client_id", clientID)
		}
	}
}
