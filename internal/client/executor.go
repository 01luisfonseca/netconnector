package client

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"netconnector/internal/shared/httpconv"
	"netconnector/pkg/logger"
	pb "netconnector/proto/pb"
)

type Executor struct {
	localAppURL string
	httpClient  *http.Client

	// Stream serialization
	sendMu sync.Mutex
}

func NewExecutor(localAppURL string) *Executor {
	return &Executor{
		localAppURL: localAppURL,
		httpClient: &http.Client{
			// Absolute timeout for local requests. No local app should take longer than 15s.
			Timeout: 15 * time.Second,
		},
	}
}

// Handle executes an incoming Protobuf HTTPRequest and writes the Response back to the stream
func (e *Executor) Handle(req *pb.HTTPRequest, stream pb.TunnelService_TunnelStreamClient) {
	reqID := req.GetRequestId()
	logger.Debug("Received request from VPS", "request_id", reqID, "method", req.GetMethod(), "url", req.GetUrl())

	// 1. Translate to Local HTTP Request
	httpReq, err := httpconv.PbToRequest(e.localAppURL, req)
	if err != nil {
		logger.Error("Failed to translate protobuf request", "err", err, "request_id", reqID)
		e.sendError(reqID, http.StatusBadRequest, stream)
		return
	}

	// 2. Execute against Local App
	httpResp, err := e.httpClient.Do(httpReq)
	if err != nil {
		logger.Error("Local App execution failed", "err", err, "request_id", reqID)
		// Usually indicates local server is down or timed out
		e.sendError(reqID, http.StatusBadGateway, stream)
		return
	}
	defer httpResp.Body.Close()

	// 3. Translate Local HTTP Response to Protobuf
	pbResp, err := httpconv.ResponseToPb(reqID, httpResp)
	if err != nil {
		logger.Error("Failed to translate local response to protobuf", "err", err, "request_id", reqID)
		e.sendError(reqID, http.StatusInternalServerError, stream)
		return
	}

	// 4. Send back to VPS
	e.sendResponse(pbResp, stream)
}

func (e *Executor) sendError(reqID string, statusCode int, stream pb.TunnelService_TunnelStreamClient) {
	resp := &pb.HTTPResponse{
		RequestId:  reqID,
		StatusCode: int32(statusCode),
		Body:       []byte(fmt.Sprintf("Local Executor Error: %d", statusCode)),
		Headers: map[string]*pb.HeaderList{
			"Content-Type": {Values: []string{"text/plain"}},
		},
	}
	e.sendResponse(resp, stream)
}

// Thread-safe wrapper for gRPC stream.Send
func (e *Executor) sendResponse(res *pb.HTTPResponse, stream pb.TunnelService_TunnelStreamClient) {
	e.sendMu.Lock()
	defer e.sendMu.Unlock()

	msg := &pb.TunnelMessage{
		Payload: &pb.TunnelMessage_HttpResponse{
			HttpResponse: res,
		},
	}

	err := stream.Send(msg)
	if err != nil {
		logger.Error("Failed to send response back to VPS", "err", err, "request_id", res.GetRequestId())
	} else {
		logger.Debug("Response sent", "request_id", res.GetRequestId(), "status", res.GetStatusCode())
	}
}
