package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"os"

	pb "tomshop/grpc"
	repo "tomshop/repositories/sql"
	"tomshop/services"

	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	_ "github.com/lib/pq"
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

	s := grpc.NewServer(grpc.UnaryInterceptor(grpc_recovery.UnaryServerInterceptor()))

	db, err := sql.Open("postgres", os.Getenv("DATABASE_ADDR"))
	if err != nil {
		log.Fatal("error connecting to the database: ", err)
	}

	pb.RegisterTomShopServer(s, services.NewOrderService(func(ctx context.Context) services.Repo {
		return repo.NewCockroachRepo(ctx, db)
	}))

	health.RegisterHealthServer(s, &services.HealthcheckService{})

	log.Println("GRPC server listening on ", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
