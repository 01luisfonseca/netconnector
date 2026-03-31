package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	pb "netconnector/proto/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// MockStream implements pb.TunnelService_TunnelStreamClient for testing
type MockStream struct {
	grpc.ClientStream
	SentMessages []*pb.TunnelMessage
}

func (m *MockStream) Send(msg *pb.TunnelMessage) error {
	m.SentMessages = append(m.SentMessages, msg)
	return nil
}

func (m *MockStream) Recv() (*pb.TunnelMessage, error) {
	return nil, nil // Not used in this test
}

func (m *MockStream) CloseSend() error {
	return nil
}

func (m *MockStream) Context() context.Context {
	return context.Background()
}

func (m *MockStream) Header() (metadata.MD, error) {
	return nil, nil
}

func (m *MockStream) Trailer() metadata.MD {
	return nil
}

func (m *MockStream) SendMsg(msg any) error {
	return nil
}

func (m *MockStream) RecvMsg(msg any) error {
	return nil
}

func TestExecutor_Handle(t *testing.T) {
	// 1. Setup a dummy local app
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Echo-Method", r.Method)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("hello from local app"))
	}))
	defer server.Close()

	executor := NewExecutor(server.URL)
	stream := &MockStream{}

	// 2. Simulate an incoming request from VPS
	req := &pb.HTTPRequest{
		RequestId: "req-123",
		Method:    http.MethodPost,
		Url:       "/test-path",
		Body:      []byte("payload"),
	}

	executor.Handle(req, stream)

	// 3. Verify results
	if len(stream.SentMessages) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(stream.SentMessages))
	}

	pbResp := stream.SentMessages[0].GetHttpResponse()
	if pbResp == nil {
		t.Fatal("expected an HTTP response payload")
	}

	if pbResp.RequestId != "req-123" {
		t.Errorf("expected RequestID req-123, got %s", pbResp.RequestId)
	}

	if pbResp.StatusCode != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", pbResp.StatusCode)
	}

	if string(pbResp.Body) != "hello from local app" {
		t.Errorf("expected body 'hello from local app', got %s", string(pbResp.Body))
	}

	if pbResp.Headers["X-Echo-Method"].Values[0] != "POST" {
		t.Errorf("expected header X-Echo-Method: POST, got %v", pbResp.Headers["X-Echo-Method"].Values)
	}
}

func TestExecutor_HandleError(t *testing.T) {
	// Point to a non-existent server to trigger StatusBadGateway
	executor := NewExecutor("http://localhost:9999")
	stream := &MockStream{}

	req := &pb.HTTPRequest{
		RequestId: "req-fail",
		Method:    http.MethodGet,
		Url:       "/",
	}

	executor.Handle(req, stream)

	if len(stream.SentMessages) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(stream.SentMessages))
	}

	pbResp := stream.SentMessages[0].GetHttpResponse()
	if pbResp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", pbResp.StatusCode)
	}
}
