package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"netconnector/internal/shared/httpconv"
	"netconnector/pkg/logger"

	"github.com/google/uuid"
)

// ProxyHandler implements http.Handler, acting as the bridge from Internet to gRPC.
type ProxyHandler struct {
	db     Repository
	router *Router
}

func NewProxyHandler(db Repository, router *Router) *ProxyHandler {
	return &ProxyHandler{
		db:     db,
		router: router,
	}
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Extract Host (Subdomain lookup)
	host := r.Host // e.g. mi-app.midominio.com
	subdomain := extractSubdomain(host)

	// 2. Fetch Client ID from SQLite
	clientID, err := h.db.GetClientIDBySubdomain(subdomain)
	if err != nil {
		logger.Error("Database lookup failed for subdomain", "subdomain", subdomain, "err", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	if clientID == "" {
		http.Error(w, fmt.Sprintf("502 Bad Gateway: Unmapped subdomain %s", subdomain), http.StatusBadGateway)
		return
	}

	// 3. Check if Client is connected (in memory router)
	client, err := h.router.GetClient(clientID)
	if err != nil {
		http.Error(w, "502 Bad Gateway: Local agent offline", http.StatusBadGateway)
		return
	}

	// 4. Translate HTTP Request to Protobuf
	reqID := uuid.New().String()
	pbReq, err := httpconv.RequestToPb(reqID, r)
	if err != nil {
		logger.Error("Failed to translate HTTP to Protobuf", "err", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 5. Register Promise in Router (buffered channel)
	respChan := client.RegisterPromise(reqID)
	// Make sure we cancel it if we timeout or client disconnects
	defer client.CancelPromise(reqID)

	// 6. Send Request over gRPC
	if err := client.SendRequest(pbReq); err != nil {
		logger.Error("Failed to send request to agent", "client_id", clientID, "err", err)
		http.Error(w, "502 Bad Gateway: Failed to reach agent", http.StatusBadGateway)
		return
	}

	// 7. Wait for Response (or Timeout)
	// We set a hard timeout of e.g., 30s to prevent hanging the HTTP handler forever
	timeout := time.After(30 * time.Second)

	select {
	case pbResp, ok := <-respChan:
		if !ok {
			// Channel was closed without response
			http.Error(w, "502 Bad Gateway: Agent disconnected during request", http.StatusBadGateway)
			return
		}

		// Translate Protobuf Response back to HTTP
		httpconv.PbToResponse(pbResp, w)
		return
	case <-timeout:
		// Client agent took too long
		logger.Warn("Request timeout", "request_id", reqID, "client_id", clientID)
		http.Error(w, "504 Gateway Timeout: Local app took too long", http.StatusGatewayTimeout)
		return
	case <-r.Context().Done():
		// External client (Browser/Nginx) disconnected early
		logger.Warn("Client disconnected early", "request_id", reqID, "client_id", clientID)
		return
	}
}

// Helpers

func extractSubdomain(host string) string {
	// host could be `sub.domain.com` or `sub.domain.com:8080`.
	// For simplicity, we split on colon if present and grab the first part (or extract properly).
	cleanHost := strings.Split(host, ":")[0]
	// If domain is "mi-app.ngrok.io", subdomain is "mi-app". You might want to match full Host instead.
	// For this system, we'll assume the full clean Host is the "subdomain"/key we map in SQLite.
	return cleanHost
}
