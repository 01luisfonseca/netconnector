package main

import (
	"log"
	"net"
	"net/http"
	"os"

	"time"

	"netconnector/internal/server"
	"netconnector/pkg/logger"
	"github.com/joho/godotenv"
	pb "netconnector/proto/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		logger.Debug("No .env file found, relying on environment variables")
	}

	// Simple config via ENV
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./mappings.db" // In cwd
	}

	logger.Info("Starting Netconnector Server (VPS)", "grpc_port", grpcPort, "http_port", httpPort)

	// Dependency Injection
	db, err := server.NewSQLiteDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	router := server.NewRouter()

	// 1. Start gRPC Server (The Bridge to Agents)
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("failed to listen on gRPC port: %v", err)
	}

	// KeepAlive to clean up dead connections gracefully
	kepParams := grpc.KeepaliveParams(keepalive.ServerParameters{
		Time:    30 * time.Second, // Send ping every 30s
		Timeout: 10 * time.Second, // If ping not answered in 10s, close
	})

	// Allow clients to send keepalive pings more frequently (needed for long-lived streams)
	kepPolicy := grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second, // Allow pings as often as every 5s
		PermitWithoutStream: true,            // Allow pings even when there's no active request
	})

	tlsCertFile := os.Getenv("TLS_CERT_FILE")
	if tlsCertFile == "" {
		if _, err := os.Stat("certs/server.crt"); err == nil {
			tlsCertFile = "certs/server.crt"
		}
	}

	tlsKeyFile := os.Getenv("TLS_KEY_FILE")
	if tlsKeyFile == "" {
		if _, err := os.Stat("certs/server.key"); err == nil {
			tlsKeyFile = "certs/server.key"
		}
	}

	var grpcServer *grpc.Server
	if tlsCertFile != "" && tlsKeyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(tlsCertFile, tlsKeyFile)
		if err != nil {
			log.Fatalf("Failed to generate server TLS credentials: %v", err)
		}
		logger.Info("gRPC TLS Enabled", "cert", tlsCertFile)
		grpcServer = grpc.NewServer(kepParams, kepPolicy, grpc.Creds(creds))
	} else {
		logger.Warn("gRPC TLS Disabled. Running in insecure mode.")
		grpcServer = grpc.NewServer(kepParams, kepPolicy)
	}

	pb.RegisterTunnelServiceServer(grpcServer, server.NewTunnelServer(router, db))

	go func() {
		logger.Info("gRPC server listening", "address", lis.Addr())
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	// 2. Start HTTP Server (The Internet endpoint)
	proxyHandler := server.NewProxyHandler(db, router)
	httpMux := http.NewServeMux()
	httpMux.Handle("/", proxyHandler)

	logger.Info("HTTP server listening", "address", ":"+httpPort)
	if err := http.ListenAndServe(":"+httpPort, httpMux); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}
