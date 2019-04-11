package sql

import (
	"context"
	"database/sql"
	"fmt"

	"tomshop/repositories"

	"github.com/cockroachdb/cockroach-go/crdb"
	"github.com/lib/pq"
)

// CockroachRepo built for CockroachDB in mind but can worl pretty well with any SQL DBMS
type CockroachRepo struct {
	txnFactory func(context.Context, *sql.TxOptions) (crdb.Tx, error)
	querier    Querier
}

// NewCockroachRepo with sql.DB, ctx must be request scope
func NewCockroachRepo(db *sql.DB) *CockroachRepo {
	return &CockroachRepo{
		txnFactory: func(ctx context.Context, opts *sql.TxOptions) (crdb.Tx, error) {
			return db.BeginTx(ctx, opts)
		},
		querier: db,
	}
}

// AdjustInventories uses Quantity as delta, not absolute value
func (r *CockroachRepo) AdjustInventories(ctx context.Context, orders []repositories.Order) error {
	if len(orders) == 0 {
		return nil
	}

	tx, err := r.txnFactory(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}

	return crdb.ExecuteInTx(ctx, tx, func() error {
		updateStmt := "UPDATE inventories SET stock_count = stock_count - $1 WHERE id = $2 AND stock_count >= $3"
		for _, o := range orders {
			result, err := tx.ExecContext(ctx, updateStmt, o.Quantity, o.ProductID, o.Quantity)
			if err != nil {
				return err
			}

			n, err := result.RowsAffected()
			if err != nil {
				return err
			}

			if n == 0 {
				return &inventoryAdjustError{
					error:     fmt.Errorf("cannot modify stock quantity for product %d", o.ProductID),
					productID: o.ProductID,
				}
			}
		}

		return nil
	})

}

// ListInventories by ID, omit items that not in DB
func (r *CockroachRepo) ListInventories(ctx context.Context, IDs []int64) ([]repositories.Inventory, error) {
	rows, err := r.querier.QueryContext(ctx, "SELECT id, stock_count FROM inventories WHERE id = ANY ($1)", pq.Array(IDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]repositories.Inventory, 0, len(IDs))
	for rows.Next() {
		inv := repositories.Inventory{}
		if err := rows.Scan(&inv.ProductID, &inv.StockCount); err != nil {
			return nil, err
		}
		results = append(results, inv)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// Querier implemented by sql.Stmt
type Querier interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
}

type inventoryAdjustError struct {
	error
	productID int64
}

func (e *inventoryAdjustError) ProductID() int64 {
	return e.productID
}
