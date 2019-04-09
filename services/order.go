package services

import (
	"context"

	pb "tomshop/grpc"
)

// OrderService implements grpc tomshop.v1.TomShop service
type OrderService struct{}

// MakeOrder simply rely on repository
func (s *OrderService) MakeOrder(ctx context.Context, in *pb.OrderRequest) (*pb.OrderResponse, error) {
	return &pb.OrderResponse{Successful: true}, nil
}
