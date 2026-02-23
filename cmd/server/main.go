package main

import (
	"context"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/bansal247/catalog-proj/internal/services"
	pb "github.com/bansal247/catalog-proj/internal/transport/grpc/product/pb"
)

func main() {
	ctx := context.Background()

	cfg := services.Config{
		SpannerProject:  getEnv("SPANNER_PROJECT", "test-project"),
		SpannerInstance: getEnv("SPANNER_INSTANCE", "test-instance"),
		SpannerDatabase: getEnv("SPANNER_DATABASE", "catalog"),
	}

	svc, err := services.Build(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to build services: %v", err)
	}
	defer svc.SpannerClient.Close()

	addr := getEnv("GRPC_ADDR", ":50051")
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterProductServiceServer(grpcServer, svc.ProductHandler)
	reflection.Register(grpcServer) // enables grpcurl discovery

	log.Printf("gRPC server listening on %s", addr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server error: %v", err)
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}