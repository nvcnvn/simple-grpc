package sql

import (
	"context"
	"database/sql"
	"fmt"

	"tomshop/repositories"
)

// CockroachRepo built for CockroachDB in mind but can worl pretty well with any SQL DBMS
type CockroachRepo struct {
	txnFactory  func(*sql.TxOptions) (TransactionManager, error)
	stmtFactory func(TransactionManager, string) (Executor, error)
	ctx         context.Context
}

// NewCockroachRepo with sql.DB, ctx must be request scope
func NewCockroachRepo(ctx context.Context, db *sql.DB) *CockroachRepo {
	return &CockroachRepo{
		txnFactory: func(opts *sql.TxOptions) (TransactionManager, error) {
			return db.BeginTx(ctx, opts)
		},
		stmtFactory: func(tm TransactionManager, query string) (Executor, error) {
			return tm.(*sql.Tx).PrepareContext(ctx, query)
		},
		ctx: ctx,
	}
}

// AdjustInventories checks optimistic lock using Inventory.Version and will not commit changes
// if any item failed to store
func (r *CockroachRepo) AdjustInventories(inventories []repositories.Inventory) error {
	tx, err := r.txnFactory(&sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}

	var (
		txnErr error
		stmt   Executor
	)
	stmt, txnErr = r.stmtFactory(tx, "UPDATE repositories SET stock_count = ? WHERE id = ? AND version = ?")
	if err != nil {
		return txnErr
	}
	defer func() {
		stmt.Close()
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			txnErr = tx.Commit()
		}
	}()

	// update one by one... yes
	for _, inventory := range inventories {
		var result sql.Result
		result, txnErr = stmt.ExecContext(r.ctx, inventory.ProductID, inventory.StockCount, inventory.Version)
		if txnErr != nil {
			return txnErr
		}

		var n int64
		n, txnErr = result.RowsAffected()
		if txnErr != nil {
			return txnErr
		}

		if n == 0 {
			txnErr = fmt.Errorf("cannot modify stock quantity for product %d", inventory.ProductID)
			return txnErr
		}
	}

	return txnErr
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
