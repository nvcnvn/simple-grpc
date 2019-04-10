package services

import (
	"context"
	"fmt"
	"testing"

	pb "tomshop/grpc"
	"tomshop/repositories"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestOrderService_MakeOrder(t *testing.T) {
	t.Run("expecting gRPC Internal error if got error when ListInventories",
		errorWhenListInventories)
	t.Run("expecting gRPC Internal error if got error when AdjustInventories",
		errorWhenAdjustInventories)
	t.Run("expecting gRPC FailedPrecondition error if don't have enough products",
		errorWhenAvailableInventoriesMissingProduct)
	t.Run("expecting gRPC FailedPrecondition error if products don't have enough items",
		errorWhenAvailableInventoriesNotEnough)
}

func errorWhenListInventories(t *testing.T) {
	s := &OrderService{
		Repo: mockRepo{
			listInventories: func(context.Context, []int64) ([]repositories.Inventory, error) {
				return nil, fmt.Errorf("dummyListInventoriesError")
			},
		},
	}

	resp, err := s.MakeOrder(context.Background(), &pb.OrderRequest{
		Purchases: []*pb.Order{
			{
				ProductID: 1,
				Quantity:  11,
			},
		},
	})

	if status.Code(err) != codes.Internal {
		t.Error("expecting gRPC Internal error, got", err)
	}

	if resp.Successful {
		t.Error("expecting failed response, got", resp)
	}
}

func errorWhenAdjustInventories(t *testing.T) {
	s := &OrderService{
		Repo: mockRepo{
			listInventories: func(context.Context, []int64) ([]repositories.Inventory, error) {
				return []repositories.Inventory{
					{
						ProductID:  1,
						StockCount: 11,
						Version:    111,
					},
				}, nil
			},
			adjustInventories: func([]repositories.Inventory) error {
				return fmt.Errorf("dummyAdjustInventoriesError")
			},
		},
	}

	resp, err := s.MakeOrder(context.Background(), &pb.OrderRequest{
		Purchases: []*pb.Order{
			{
				ProductID: 1,
				Quantity:  11,
			},
		},
	})

	if status.Code(err) != codes.Internal {
		t.Error("expecting gRPC Internal error, got", err)
	}

	if resp.Successful {
		t.Error("expecting failed response, got", resp)
	}
}

func errorWhenAvailableInventoriesMissingProduct(t *testing.T) {
	s := &OrderService{
		Repo: mockRepo{
			listInventories: func(context.Context, []int64) ([]repositories.Inventory, error) {
				return []repositories.Inventory{
					{
						ProductID:  1,
						StockCount: 11,
						Version:    111,
					},
				}, nil
			},
		},
	}

	resp, err := s.MakeOrder(context.Background(), &pb.OrderRequest{
		Purchases: []*pb.Order{
			{
				ProductID: 1,
				Quantity:  11,
			},
			{
				ProductID: 2,
				Quantity:  22,
			},
		},
	})

	if status.Code(err) != codes.FailedPrecondition {
		t.Error("expecting gRPC FailedPrecondition error, got", err)
	}

	if resp.Successful {
		t.Error("expecting failed response, got", resp)
	}
}

func errorWhenAvailableInventoriesNotEnough(t *testing.T) {
	s := &OrderService{
		Repo: mockRepo{
			listInventories: func(context.Context, []int64) ([]repositories.Inventory, error) {
				return []repositories.Inventory{
					{
						ProductID:  1,
						StockCount: 11,
						Version:    111,
					},
					{
						ProductID:  2,
						StockCount: 21,
						Version:    222,
					},
				}, nil
			},
		},
	}

	resp, err := s.MakeOrder(context.Background(), &pb.OrderRequest{
		Purchases: []*pb.Order{
			{
				ProductID: 1,
				Quantity:  11,
			},
			{
				ProductID: 2,
				Quantity:  22,
			},
		},
	})

	if status.Code(err) != codes.FailedPrecondition {
		t.Error("expecting gRPC FailedPrecondition error, got", err)
	}

	if resp.Successful {
		t.Error("expecting failed response, got", resp)
	}
}

type mockRepo struct {
	listInventories   func(context.Context, []int64) ([]repositories.Inventory, error)
	adjustInventories func([]repositories.Inventory) error
}

func (r mockRepo) ListInventories(ctx context.Context, ids []int64) ([]repositories.Inventory, error) {
	return r.listInventories(ctx, ids)
}

func (r mockRepo) AdjustInventories(i []repositories.Inventory) error {
	return r.adjustInventories(i)
}
