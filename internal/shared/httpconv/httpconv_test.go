package httpconv

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	pb "netconnector/proto/pb"
)

func TestRequestToPb(t *testing.T) {
	reqID := "test-req-id"
	body := []byte("hello world")
	r := httptest.NewRequest(http.MethodPost, "/foo/bar?baz=qux", bytes.NewReader(body))
	r.Header.Set("X-Test", "value")

	pbReq, err := RequestToPb(reqID, r)
	if err != nil {
		t.Fatalf("RequestToPb failed: %v", err)
	}

	if pbReq.RequestId != reqID {
		t.Errorf("expected RequestId %s, got %s", reqID, pbReq.RequestId)
	}
	if pbReq.Method != http.MethodPost {
		t.Errorf("expected Method %s, got %s", http.MethodPost, pbReq.Method)
	}
	if pbReq.Url != "/foo/bar?baz=qux" {
		t.Errorf("expected Url /foo/bar?baz=qux, got %s", pbReq.Url)
	}
	if string(pbReq.Body) != string(body) {
		t.Errorf("expected Body %s, got %s", body, pbReq.Body)
	}
	if pbReq.Headers["X-Test"].Values[0] != "value" {
		t.Errorf("expected header X-Test: value, got %v", pbReq.Headers["X-Test"].Values)
	}
}

func TestPbToResponse(t *testing.T) {
	pbResp := &pb.HTTPResponse{
		RequestId:  "test-id",
		StatusCode: 201,
		Headers: map[string]*pb.HeaderList{
			"Content-Type": {Values: []string{"application/json"}},
		},
		Body: []byte(`{"status":"ok"}`),
	}

	w := httptest.NewRecorder()
	PbToResponse(pbResp, w)

	if w.Code != 201 {
		t.Errorf("expected StatusCode 201, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}
	if w.Body.String() != `{"status":"ok"}` {
		t.Errorf("expected Body %s, got %s", `{"status":"ok"}`, w.Body.String())
	}
}

func TestPbToRequest(t *testing.T) {
	baseURL := "http://localhost:8080"
	pbReq := &pb.HTTPRequest{
		RequestId: "test-id",
		Method:    http.MethodGet,
		Url:       "/test",
		Headers: map[string]*pb.HeaderList{
			"Accept": {Values: []string{"text/plain"}},
		},
		Body: []byte("request-body"),
	}

	req, err := PbToRequest(baseURL, pbReq)
	if err != nil {
		t.Fatalf("PbToRequest failed: %v", err)
	}

	if req.Method != http.MethodGet {
		t.Errorf("expected Method %s, got %s", http.MethodGet, req.Method)
	}
	if req.URL.String() != "http://localhost:8080/test" {
		t.Errorf("expected URL http://localhost:8080/test, got %s", req.URL.String())
	}
	if req.Header.Get("Accept") != "text/plain" {
		t.Errorf("expected header Accept: text/plain, got %s", req.Header.Get("Accept"))
	}

	body, _ := io.ReadAll(req.Body)
	if string(body) != "request-body" {
		t.Errorf("expected body request-body, got %s", string(body))
	}
}

func TestResponseToPb(t *testing.T) {
	reqID := "test-id"
	r := &http.Response{
		StatusCode: 200,
		Header: http.Header{
			"X-Custom": []string{"custom-value"},
		},
		Body: io.NopCloser(bytes.NewReader([]byte("response-payload"))),
	}

	pbResp, err := ResponseToPb(reqID, r)
	if err != nil {
		t.Fatalf("ResponseToPb failed: %v", err)
	}

	if pbResp.RequestId != reqID {
		t.Errorf("expected RequestId %s, got %s", reqID, pbResp.RequestId)
	}
	if pbResp.StatusCode != 200 {
		t.Errorf("expected StatusCode 200, got %d", pbResp.StatusCode)
	}
	if pbResp.Headers["X-Custom"].Values[0] != "custom-value" {
		t.Errorf("expected header X-Custom: custom-value, got %v", pbResp.Headers["X-Custom"].Values)
	}
	if string(pbResp.Body) != "response-payload" {
		t.Errorf("expected Body response-payload, got %s", string(pbResp.Body))
	}
}

func TestRequestToPbNoBody(t *testing.T) {
	reqID := "test-req-id"
	r := httptest.NewRequest(http.MethodGet, "/foo", nil)

	pbReq, err := RequestToPb(reqID, r)
	if err != nil {
		t.Fatalf("RequestToPb failed: %v", err)
	}

	if len(pbReq.Body) != 0 {
		t.Errorf("expected empty body, got length %d", len(pbReq.Body))
	}
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("read error")
}

func (e *errorReader) Close() error { return nil }

func TestRequestToPbReadError(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/foo", &errorReader{})
	_, err := RequestToPb("test", r)
	if err == nil {
		t.Error("expected error due to read failure")
	}
}
