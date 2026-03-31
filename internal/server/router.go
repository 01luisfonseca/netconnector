package server

import (
	"fmt"
	"netconnector/pkg/logger"
	pb "netconnector/proto/pb"
	"sync"
)

// ActiveClient represents a connected Local Agent.
type ActiveClient struct {
	clientID string
	stream   pb.TunnelService_TunnelStreamServer
	// Protects concurrent calls to stream.Send
	// Although Send is thread-safe conceptually in some languages,
	// grpc-go documentation states: "stream RPCs can be called concurrently,
	// but it is not safe to call Send or Recv concurrently from multiple goroutines on the same stream."
	// Since we might be sending multiple requests concurrently to a single client stream, we need a mutex for Send.
	sendMu sync.Mutex

	// Map of request_id to a channel expecting the HTTPResponse
	reqsMu          sync.RWMutex
	pendingRequests map[string]chan *pb.HTTPResponse
}

func NewActiveClient(id string, stream pb.TunnelService_TunnelStreamServer) *ActiveClient {
	return &ActiveClient{
		clientID:        id,
		stream:          stream,
		pendingRequests: make(map[string]chan *pb.HTTPResponse),
	}
}

func (ac *ActiveClient) SendRequest(req *pb.HTTPRequest) error {
	ac.sendMu.Lock()
	defer ac.sendMu.Unlock()

	msg := &pb.TunnelMessage{
		Payload: &pb.TunnelMessage_HttpRequest{
			HttpRequest: req,
		},
	}
	return ac.stream.Send(msg)
}

func (ac *ActiveClient) RegisterPromise(reqID string) chan *pb.HTTPResponse {
	ch := make(chan *pb.HTTPResponse, 1) // Buffer of 1 so sending doesn't block if we timed out
	ac.reqsMu.Lock()
	ac.pendingRequests[reqID] = ch
	ac.reqsMu.Unlock()
	return ch
}

func (ac *ActiveClient) ResolvePromise(reqID string, res *pb.HTTPResponse) {
	ac.reqsMu.Lock()
	ch, exists := ac.pendingRequests[reqID]
	if exists {
		delete(ac.pendingRequests, reqID)
	}
	ac.reqsMu.Unlock()

	if exists {
		ch <- res
		close(ch)
	} else {
		logger.Warn("Received response for unknown or expired request_id", "client_id", ac.clientID, "request_id", reqID)
	}
}

func (ac *ActiveClient) CancelPromise(reqID string) {
	ac.reqsMu.Lock()
	ch, exists := ac.pendingRequests[reqID]
	if exists {
		delete(ac.pendingRequests, reqID)
		close(ch)
	}
	ac.reqsMu.Unlock()
}

// Router manages the global state of connected clients.
type Router struct {
	mu      sync.RWMutex
	clients map[string]*ActiveClient
}

func NewRouter() *Router {
	return &Router{
		clients: make(map[string]*ActiveClient),
	}
}

func (r *Router) RegisterClient(clientID string, stream pb.TunnelService_TunnelStreamServer) (*ActiveClient, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.clients[clientID]; exists {
		logger.Warn("Client re-registered, terminating previous connection if any", "client_id", clientID)
		// We overwrite it, the previous goroutine will eventually error out and exit cleanup.
		// A more robust implementation might explicitly terminate the old context.
	}

	ac := NewActiveClient(clientID, stream)
	r.clients[clientID] = ac
	logger.Info("Client registered successfully", "client_id", clientID)
	return ac, nil
}

func (r *Router) UnregisterClient(clientID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.clients[clientID]; exists {
		delete(r.clients, clientID)
		logger.Info("Client unregistered", "client_id", clientID)
	}
}

func (r *Router) GetClient(clientID string) (*ActiveClient, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	client, exists := r.clients[clientID]
	if !exists {
		return nil, fmt.Errorf("client not connected: %s", clientID)
	}
	return client, nil
}
