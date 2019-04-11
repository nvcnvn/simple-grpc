package services

import (
	"context"
	"log"

	pb "tomshop/grpc"
	"tomshop/repositories"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	notEnoughStockErr  = status.Error(codes.FailedPrecondition, "not enough stock to fullfil order")
	invalidOrderQtyErr = status.Error(codes.InvalidArgument, "invalid quantity for order")
)

// OrderService implements grpc tomshop.v1.TomShop service
type OrderService struct {
	Repo interface {
		ListInventories(context.Context, []int64) ([]repositories.Inventory, error)
		AdjustInventories(context.Context, []repositories.Order) error
	}
}

// MakeOrder simply rely on repository
func (s *OrderService) MakeOrder(ctx context.Context, in *pb.OrderRequest) (*pb.OrderResponse, error) {
	ids := make([]int64, len(in.Purchases))
	purchaseMap := make(map[int64]int64, len(in.Purchases))
	for i, order := range in.Purchases {
		ids[i] = order.ProductID
		purchaseMap[order.ProductID] = order.Quantity
	}

	availableInventories, err := s.Repo.ListInventories(ctx, ids)
	if err != nil {
		return &pb.OrderResponse{
			Successful: false,
		}, status.Errorf(codes.Internal, "internal error when checking for available stock: %s", err.Error())
	}

	if len(availableInventories) != len(in.Purchases) {
		log.Println("not enough products")
		return &pb.OrderResponse{
			Successful: false,
		}, notEnoughStockErr
	}

	orders := make([]repositories.Order, len(availableInventories))

	for i := range availableInventories {
		productID := availableInventories[i].ProductID
		requestQty := purchaseMap[productID]
		if requestQty <= 0 {
			return &pb.OrderResponse{
				Successful: false,
			}, invalidOrderQtyErr
		}

		if requestQty > availableInventories[i].StockCount {
			log.Printf(
				"not enough items for product %d, reuested: %d, available: %d",
				productID,
				requestQty,
				availableInventories[i].StockCount,
			)
			return &pb.OrderResponse{
				Successful: false,
			}, notEnoughStockErr
		}

		orders[i].ProductID = productID
		orders[i].Quantity = requestQty
	}

	err = s.Repo.AdjustInventories(ctx, orders)
	if err != nil {
		return &pb.OrderResponse{
			Successful: false,
		}, status.Errorf(codes.Internal, "internal error when saving order: %s", err.Error())
	}

	return &pb.OrderResponse{Successful: true}, nil
}
