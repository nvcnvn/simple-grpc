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
	notEnoughStockErr = status.Error(codes.FailedPrecondition, "not enough stock to fullfil order")
)

// OrderService implements grpc tomshop.v1.TomShop service
type OrderService struct {
	repoFactory func(ctx context.Context) Repo
}

// Repo implemented by repositories/sql
type Repo interface {
	ListInventories(context.Context, []int64) ([]repositories.Inventory, error)
	AdjustInventories([]repositories.Inventory) error
}

// NewOrderService simply returns new OrderService
func NewOrderService(repoFactory func(ctx context.Context) Repo) *OrderService {
	return &OrderService{
		repoFactory: repoFactory,
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

	repo := s.repoFactory(ctx)
	availableInventories, err := repo.ListInventories(ctx, ids)
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

	for i := range availableInventories {
		requestQty := purchaseMap[availableInventories[i].ProductID]
		if requestQty > availableInventories[i].StockCount {
			log.Printf(
				"not enough items for product %d, reuested: %d, available: %d",
				availableInventories[i].ProductID,
				requestQty,
				availableInventories[i].StockCount,
			)
			return &pb.OrderResponse{
				Successful: false,
			}, notEnoughStockErr
		}
		availableInventories[i].StockCount -= requestQty
	}

	err = repo.AdjustInventories(availableInventories)
	if err != nil {
		return &pb.OrderResponse{
			Successful: false,
		}, status.Errorf(codes.Internal, "internal error when saving order: %s", err.Error())
	}

	return &pb.OrderResponse{Successful: true}, nil
}
