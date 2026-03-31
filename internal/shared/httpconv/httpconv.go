package httpconv

import (
	"bytes"
	"io"
	"net/http"

	pb "netconnector/proto/pb"
)

// RequestToPb translates an incoming HTTP Request to a Protobuf HTTPRequest
func RequestToPb(reqID string, r *http.Request) (*pb.HTTPRequest, error) {
	var bodyBytes []byte
	var err error
	if r.Body != nil {
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		r.Body.Close()
	}

	pbHeaders := make(map[string]*pb.HeaderList)
	for k, vv := range r.Header {
		pbHeaders[k] = &pb.HeaderList{Values: vv}
	}

	return &pb.HTTPRequest{
		RequestId: reqID,
		Method:    r.Method,
		Url:       r.RequestURI, // Path + Query string
		Headers:   pbHeaders,
		Body:      bodyBytes,
	}, nil
}

// PbToResponse translates a Protobuf HTTPResponse to populate an http.ResponseWriter
func PbToResponse(pbResp *pb.HTTPResponse, w http.ResponseWriter) {
	for k, hList := range pbResp.GetHeaders() {
		for _, v := range hList.GetValues() {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(int(pbResp.GetStatusCode()))
	if len(pbResp.GetBody()) > 0 {
		_, _ = w.Write(pbResp.GetBody())
	}
}

// PbToRequest translates a Protobuf HTTPRequest to an outgoing http.Request (useful for the local Agent)
func PbToRequest(baseURL string, pbReq *pb.HTTPRequest) (*http.Request, error) {
	fullURL := baseURL + pbReq.GetUrl()

	var bodyReader io.Reader
	if len(pbReq.GetBody()) > 0 {
		bodyReader = bytes.NewReader(pbReq.GetBody())
	}

	req, err := http.NewRequest(pbReq.GetMethod(), fullURL, bodyReader)
	if err != nil {
		return nil, err
	}

	for k, hList := range pbReq.GetHeaders() {
		for _, v := range hList.GetValues() {
			req.Header.Add(k, v)
		}
	}

	return req, nil
}

// ResponseToPb translates an http.Response to a Protobuf HTTPResponse (useful for the local Agent)
func ResponseToPb(reqID string, r *http.Response) (*pb.HTTPResponse, error) {
	var bodyBytes []byte
	var err error
	if r.Body != nil {
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		r.Body.Close()
	}

	pbHeaders := make(map[string]*pb.HeaderList)
	for k, vv := range r.Header {
		pbHeaders[k] = &pb.HeaderList{Values: vv}
	}

	return &pb.HTTPResponse{
		RequestId:  reqID,
		StatusCode: int32(r.StatusCode),
		Headers:    pbHeaders,
		Body:       bodyBytes,
	}, nil
}
