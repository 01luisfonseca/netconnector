package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pb "netconnector/proto/pb"
	"google.golang.org/grpc/metadata"
)

// MockRepo implements Repository
type MockRepo struct {
	mappings map[string]string
	Err      error
}

func (m *MockRepo) GetClientIDBySubdomain(subdomain string) (string, error) {
	if m.Err != nil { return "", m.Err }
	return m.mappings[subdomain], nil
}
func (m *MockRepo) AddMapping(sub, id string) error { return nil }
func (m *MockRepo) RemoveMapping(sub string) error  { return nil }
func (m *MockRepo) ListMappings() (map[string]string, error) { return nil, nil }
func (m *MockRepo) IsClientIDRegistered(id string) (bool, error) {
	if m.Err != nil { return false, m.Err }
	for _, mappingID := range m.mappings {
		if mappingID == id {
			return true, nil
		}
	}
	return false, nil
}
func (m *MockRepo) Close() error { return nil }

// MockServerStream implements pb.TunnelService_TunnelStreamServer
type MockServerStream struct {
	SentMsg *pb.TunnelMessage
}

func (m *MockServerStream) Send(msg *pb.TunnelMessage) error {
	m.SentMsg = msg
	return nil
}
func (m *MockServerStream) Recv() (*pb.TunnelMessage, error) { return nil, nil }
func (m *MockServerStream) SetHeader(metadata.MD) error { return nil }
func (m *MockServerStream) SendHeader(metadata.MD) error { return nil }
func (m *MockServerStream) SetTrailer(metadata.MD) {}
func (m *MockServerStream) Context() context.Context { return context.Background() }
func (m *MockServerStream) SendMsg(msg any) error { return nil }
func (m *MockServerStream) RecvMsg(msg any) error { return nil }

func TestProxyHandler_ServeHTTP(t *testing.T) {
	db := &MockRepo{
		mappings: map[string]string{"test.com": "client-1"},
	}
	router := NewRouter()
	handler := NewProxyHandler(db, router)

	t.Run("UnmappedSubdomain", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://unknown.com/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadGateway {
			t.Errorf("expected 502, got %d", w.Code)
		}
	})

	t.Run("ClientOffline", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://test.com/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadGateway {
			t.Errorf("expected 502, got %d", w.Code)
		}
	})

	t.Run("SuccessFlow", func(t *testing.T) {
		stream := &MockServerStream{}
		client, _ := router.RegisterClient("client-1", stream)
		req := httptest.NewRequest("GET", "http://test.com/foo", nil)
		w := httptest.NewRecorder()

		go func() {
			for stream.SentMsg == nil {
				time.Sleep(10 * time.Millisecond)
			}
			reqID := stream.SentMsg.GetHttpRequest().GetRequestId()
			client.ResolvePromise(reqID, &pb.HTTPResponse{
				StatusCode: 200,
				Body:       []byte("ok"),
			})
		}()

		handler.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("RequestToPbError", func(t *testing.T) {
		stream := &MockServerStream{}
		_, _ = router.RegisterClient("client-1", stream)
		req := httptest.NewRequest("POST", "http://test.com/", &errorReader{})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})

	t.Run("SendRequestError", func(t *testing.T) {
		stream := &errorStream{}
		_, _ = router.RegisterClient("client-1", stream)
		req := httptest.NewRequest("GET", "http://test.com/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadGateway {
			t.Errorf("expected 502, got %d", w.Code)
		}
	})
}

type errorReader struct{}
func (e *errorReader) Read(p []byte) (n int, err error) { return 0, fmt.Errorf("read error") }

type errorStream struct {
	MockServerStream
}
func (e *errorStream) Send(msg *pb.TunnelMessage) error { return fmt.Errorf("send error") }
