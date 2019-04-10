package sql

import (
	"context"
	"database/sql"
	"fmt"

	"tomshop/repositories"

	"github.com/lib/pq"
)

// CockroachRepo built for CockroachDB in mind but can worl pretty well with any SQL DBMS
type CockroachRepo struct {
	txnFactory      func(*sql.TxOptions) (TransactionManager, error)
	executorFactory func(TransactionManager, string) (Executor, error)
	querier         Querier
	ctx             context.Context
}

// NewCockroachRepo with sql.DB, ctx must be request scope
func NewCockroachRepo(ctx context.Context, db *sql.DB) *CockroachRepo {
	return &CockroachRepo{
		txnFactory: func(opts *sql.TxOptions) (TransactionManager, error) {
			return db.BeginTx(ctx, opts)
		},
		executorFactory: func(tm TransactionManager, query string) (Executor, error) {
			return tm.(*sql.Tx).PrepareContext(ctx, query)
		},
		querier: db,
		ctx:     ctx,
	}
}

// AdjustInventories checks optimistic lock using Inventory.Version and will not commit changes
// if any item failed to store
func (r *CockroachRepo) AdjustInventories(inventories []repositories.Inventory) (txnErr error) {
	if len(inventories) == 0 {
		return nil
	}

	tx, err := r.txnFactory(&sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}

	updateStmt := "UPDATE inventories SET stock_count = $1, version = version + 1 WHERE id = $2 AND version = $3"
	var stmt Executor
	stmt, txnErr = r.executorFactory(tx, updateStmt)
	if txnErr != nil {
		return
	}
	defer func() {
		stmt.Close()
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if txnErr != nil {
			tx.Rollback()
		} else {
			txnErr = tx.Commit()
		}
	}()

	// update one by one... yes
	for _, inventory := range inventories {
		var result sql.Result
		result, txnErr = stmt.ExecContext(r.ctx, inventory.StockCount, inventory.ProductID, inventory.Version)
		if txnErr != nil {
			return
		}

		var n int64
		n, txnErr = result.RowsAffected()
		if txnErr != nil {
			return
		}

		if n == 0 {
			txnErr = &inventoryAdjustError{
				error:     fmt.Errorf("cannot modify stock quantity for product %d", inventory.ProductID),
				productID: inventory.ProductID,
			}
			return
		}
	}

	return
}

// ListInventories by ID, omit items that not in DB
func (r *CockroachRepo) ListInventories(ctx context.Context, IDs []int64) ([]repositories.Inventory, error) {
	rows, err := r.querier.QueryContext(ctx, "SELECT id, stock_count, version FROM inventories WHERE id = ANY ($1)", pq.Array(IDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]repositories.Inventory, 0, len(IDs))
	for rows.Next() {
		inv := repositories.Inventory{}
		if err := rows.Scan(&inv.ProductID, &inv.StockCount, &inv.Version); err != nil {
			return nil, err
		}
		results = append(results, inv)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// TransactionManager implemented by sql.Tx
type TransactionManager interface {
	Rollback() error
	Commit() error
}

// Executor implemented by sql.Stmt
type Executor interface {
	ExecContext(context.Context, ...interface{}) (sql.Result, error)
	Close() error
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
