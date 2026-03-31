package server

import (
	"testing"
	pb "netconnector/proto/pb"
)

func TestRouter_ResolveUnknownPromise(t *testing.T) {
	router := NewRouter()
	ac, _ := router.RegisterClient("test-c", &MockServerStream{})
	
	// Resolving an unknown promise should just log a warning and not panic
	ac.ResolvePromise("unknown-id", &pb.HTTPResponse{})
}

func TestRouter_UnregisterNonExistent(t *testing.T) {
	router := NewRouter()
	router.UnregisterClient("non-existent")
}
