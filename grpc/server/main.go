package main

import (
	"log"
	"net"
	"os"

	pb "tomshop/grpc"
	"tomshop/services"

	"google.golang.org/grpc"
	health "google.golang.org/grpc/health/grpc_health_v1"
)

var (
	port = os.Getenv("PORT") // default ":50051"
)

func main() {
	if port == "" {
		port = ":50051"
	}

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterTomShopServer(s, &services.OrderService{})
	health.RegisterHealthServer(s, &services.HealthcheckService{})

	log.Println("GRPC server listening on ", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
