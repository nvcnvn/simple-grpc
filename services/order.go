package services

import (
	"context"

	pb "tomshop/grpc"
)

type OrderService struct{}

// MakeOrder simply rely on repository
func (s *OrderService) MakeOrder(ctx context.Context, in *pb.OrderRequest) (*pb.OrderResponse, error) {
	return &pb.OrderResponse{Successful: true}, nil
}
