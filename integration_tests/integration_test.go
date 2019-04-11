package integration

import (
	"context"
	"database/sql"
	"log"
	"os"
	"reflect"
	"testing"
	"time"

	pb "tomshop/grpc"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRunner(t *testing.T) {
	conn, err := grpc.Dial(os.Getenv("APP_ADDR"), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewTomShopClient(conn)

	db := setupDB(os.Getenv("DATABASE_ADDR"))

	t.Run("senario 1", func(tt *testing.T) {
		scenario1(c, db, tt)
	})

	t.Run("senario 2: simple out of stock senario", func(tt *testing.T) {
		outOfStock(c, db, tt)
	})

	t.Run("senario 3: 2 concurent request, both can be fullfil", func(tt *testing.T) {
		concurrentRequestsBothOK(c, db, tt)
	})

	t.Run("example 4: 2 concurent request, only one can be fullfil", func(tt *testing.T) {
		concurrentRequestsOnlyOneOK(c, db, tt)
	})

	t.Run("order with negative stock", func(tt *testing.T) {
		orderWithNegativeStock(c, db, tt)
	})
}

// example 1 (details can be found in Manabie Senior Golang BE Coding Challenge)
func scenario1(c pb.TomShopClient, db *sql.DB, t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	resp, err := c.MakeOrder(ctx, &pb.OrderRequest{
		Purchases: []*pb.Order{
			&pb.Order{
				ProductID: 11,
				Quantity:  2,
			},
			&pb.Order{
				ProductID: 12,
				Quantity:  1,
			},
		},
	})

	if !reflect.DeepEqual(resp, &pb.OrderResponse{
		Successful: true,
	}) {
		t.Error("expecting successful request")
	}

	if err != nil {
		t.Error("unexpected error", err)
	}

	checkUpdatedQty(db, t, 11, 8)
	checkUpdatedQty(db, t, 12, 4)
}

// example 2
func outOfStock(c pb.TomShopClient, db *sql.DB, t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := c.MakeOrder(ctx, &pb.OrderRequest{
		Purchases: []*pb.Order{
			&pb.Order{
				ProductID: 21,
				Quantity:  2,
			},
			&pb.Order{
				ProductID: 22,
				Quantity:  6,
			},
		},
	})

	if status.Code(err) != codes.FailedPrecondition {
		t.Error("expecting gRPC FailedPrecondition error, got", err)
	}
}

func orderWithNegativeStock(c pb.TomShopClient, db *sql.DB, t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := c.MakeOrder(ctx, &pb.OrderRequest{
		Purchases: []*pb.Order{
			&pb.Order{
				ProductID: 21,
				Quantity:  -2,
			},
		},
	})

	if status.Code(err) != codes.InvalidArgument {
		t.Error("expecting gRPC InvalidArgument error, got", err)
	}
}

// example 3
func concurrentRequestsBothOK(c pb.TomShopClient, db *sql.DB, t *testing.T) {
	waitChn1, waitChn2 := make(chan bool), make(chan bool)
	var err1, err2 error
	ctx1, cancel1 := context.WithTimeout(context.Background(), time.Second)
	defer cancel1()
	go func() {
		_, err1 = c.MakeOrder(ctx1, &pb.OrderRequest{
			Purchases: []*pb.Order{
				&pb.Order{
					ProductID: 31,
					Quantity:  2,
				},
				&pb.Order{
					ProductID: 32,
					Quantity:  1,
				},
			},
		})
		waitChn1 <- true
	}()

	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	go func() {
		_, err2 = c.MakeOrder(ctx2, &pb.OrderRequest{
			Purchases: []*pb.Order{
				&pb.Order{
					ProductID: 31,
					Quantity:  1,
				},
				&pb.Order{
					ProductID: 32,
					Quantity:  2,
				},
			},
		})
		waitChn2 <- true
	}()

	<-waitChn1
	<-waitChn2

	if err1 != nil {
		t.Error("unpexteced error for call 1", err1)
	}

	if err2 != nil {
		t.Error("unpexteced error for call 2", err2)
	}

	checkUpdatedQty(db, t, 31, 7)
	checkUpdatedQty(db, t, 32, 2)
}

// example 4
func concurrentRequestsOnlyOneOK(c pb.TomShopClient, db *sql.DB, t *testing.T) {
	waitChn1, waitChn2 := make(chan bool), make(chan bool)
	var err1, err2 error
	ctx1, cancel1 := context.WithTimeout(context.Background(), time.Second)
	defer cancel1()
	go func() {
		_, err1 = c.MakeOrder(ctx1, &pb.OrderRequest{
			Purchases: []*pb.Order{
				&pb.Order{
					ProductID: 41,
					Quantity:  2,
				},
				&pb.Order{
					ProductID: 42,
					Quantity:  1,
				},
			},
		})
		waitChn1 <- true
	}()

	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	go func() {
		_, err2 = c.MakeOrder(ctx2, &pb.OrderRequest{
			Purchases: []*pb.Order{
				&pb.Order{
					ProductID: 41,
					Quantity:  1,
				},
				&pb.Order{
					ProductID: 42,
					Quantity:  5,
				},
			},
		})
		waitChn2 <- true
	}()

	<-waitChn1
	<-waitChn2

	if err1 == nil && err2 == nil {
		t.Error("expecting atleast one call failed")
	}

	if err1 != nil {
		t.Log("getting error for call 1", err1)
		checkUpdatedQty(db, t, 41, 9)
		checkUpdatedQty(db, t, 42, 0)
	}

	if err2 != nil {
		t.Log("getting error for call 2", err2)
		checkUpdatedQty(db, t, 41, 8)
		checkUpdatedQty(db, t, 42, 4)
	}
}

func checkUpdatedQty(db *sql.DB, t *testing.T, productID, expectedQty int64) {
	var currentQty int64
	err := db.QueryRow(
		"SELECT stock_count FROM inventories WHERE id = $1",
		productID,
	).Scan(&currentQty)
	if err != nil {
		t.Fatal("cannot check updated stock qty", err)
	}

	if currentQty != expectedQty {
		t.Errorf("expecting qty: %d for product %d, got %d", expectedQty, productID, currentQty)
	}
}

func setupDB(connStr string) *sql.DB {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("error connecting to the database: ", err)
	}

	_, err = db.Exec(`UPSERT INTO inventories VALUES 
		(11, 10, 0),
		(12, 5, 0),
		(21, 10, 0),
		(22, 5, 0),
		(31, 10, 0),
		(32, 5, 0),
		(41, 10, 0),
		(42, 5, 0);`)
	if err != nil {
		log.Fatal("error inserting test data to the database: ", err)
	}

	return db
}
