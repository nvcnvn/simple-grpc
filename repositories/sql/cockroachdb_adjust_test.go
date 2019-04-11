package sql

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"tomshop/repositories"

	"github.com/cockroachdb/cockroach-go/crdb"
)

func TestCockroachRepo_AdjustInventories(t *testing.T) {
	t.Run("must return error if cannot start new Txn", errorWhenStartTxn)
	t.Run("must return nil error if input are empty", inputEmptyNilError)
	t.Run("must Rollback if any error happen when calling Executor.ExecContext", errorInExecContext)
	t.Run("must Rollback even if panic in execContext", panicInExecContext)
	t.Run("must Rollback when cannot adjust any item", rollBackWhenNoRowUpdated)
}

var testOrder = []repositories.Order{
	{
		ProductID: 1,
		Quantity:  11,
	},
	{
		ProductID: 2,
		Quantity:  22,
	},
}

func errorWhenStartTxn(tt *testing.T) {
	dummyErr := fmt.Errorf("dummy error")
	r := &CockroachRepo{
		txnFactory: func(ctx context.Context, opts *sql.TxOptions) (crdb.Tx, error) {
			return nil, dummyErr
		},
	}

	if !reflect.DeepEqual(r.AdjustInventories(context.Background(), testOrder), dummyErr) {
		tt.Error("expecting error return if txnFactory cannot create new transaction")
	}
}

func inputEmptyNilError(tt *testing.T) {
	r := &CockroachRepo{
		txnFactory: func(ctx context.Context, opts *sql.TxOptions) (crdb.Tx, error) {
			return &sql.Tx{}, nil
		},
	}

	if err := r.AdjustInventories(nil, nil); err != nil {
		tt.Log(err)
		tt.Error("expecting error return nil error", err)

	}
}

func errorInExecContext(tt *testing.T) {
	dummyErr := fmt.Errorf("dummy execContext error")
	type expectingCall struct {
		expecting int
		called    int
	}
	expectedFnCall := map[string]*expectingCall{
		"commit": &expectingCall{
			expecting: 0,
		},
		"rollback": &expectingCall{
			expecting: 1,
		},
		"execContext": &expectingCall{
			expecting: 1,
		},
	}

	r := &CockroachRepo{
		txnFactory: func(c context.Context, opts *sql.TxOptions) (crdb.Tx, error) {
			return mockTx{
				commit: func() error {
					expectedFnCall["commit"].called++
					return nil
				},
				rollback: func() error {
					expectedFnCall["rollback"].called++
					return nil
				},
				execContext: func(c context.Context, q string, args ...interface{}) (sql.Result, error) {
					expectedFnCall["execContext"].called++
					return nil, dummyErr
				},
			}, nil
		},
	}

	if err := r.AdjustInventories(nil, testOrder); !reflect.DeepEqual(err, dummyErr) {
		tt.Error("expecting dummyErr, got", err)
	}

	for k, v := range expectedFnCall {
		if v.called != v.expecting {
			tt.Errorf("expecting calling %s only %d time, got %d", k, v.expecting, v.called)
		}
	}
}

func panicInExecContext(tt *testing.T) {
	defer func() {
		if p := recover(); p.(string) != "dummyPannic" {
			tt.Error("expecting dummyPannic panic, got", p)
		}
	}()

	dummyErr := fmt.Errorf("dummy execContext error")
	type expectingCall struct {
		expecting int
		called    int
	}
	expectedFnCall := map[string]*expectingCall{
		"commit": &expectingCall{
			expecting: 0,
		},
		"rollback": &expectingCall{
			expecting: 1,
		},
		"execContext": &expectingCall{
			expecting: 1,
		},
	}

	r := &CockroachRepo{
		txnFactory: func(c context.Context, opts *sql.TxOptions) (crdb.Tx, error) {
			return mockTx{
				commit: func() error {
					expectedFnCall["commit"].called++
					return nil
				},
				rollback: func() error {
					expectedFnCall["rollback"].called++
					return nil
				},
				execContext: func(c context.Context, q string, args ...interface{}) (sql.Result, error) {
					expectedFnCall["execContext"].called++
					panic("dummyPannic")
				},
			}, nil
		},
	}

	if err := r.AdjustInventories(nil, testOrder); !reflect.DeepEqual(err, dummyErr) {
		tt.Error("expecting dummyErr, got", err)
	}

	for k, v := range expectedFnCall {
		if v.called != v.expecting {
			tt.Errorf("expecting calling %s only %d time, got %d", k, v.expecting, v.called)
		}
	}
}

func rollBackWhenNoRowUpdated(tt *testing.T) {
	type expectingCall struct {
		expecting int
		called    int
	}
	expectedFnCall := map[string]*expectingCall{
		"commit": &expectingCall{
			expecting: 0,
		},
		"rollback": &expectingCall{
			expecting: 1,
		},
		"execContext": &expectingCall{
			expecting: 3,
		},
	}

	r := &CockroachRepo{
		txnFactory: func(c context.Context, opts *sql.TxOptions) (crdb.Tx, error) {
			return mockTx{
				commit: func() error {
					expectedFnCall["commit"].called++
					return nil
				},
				rollback: func() error {
					expectedFnCall["rollback"].called++
					return nil
				},
				execContext: func(c context.Context, q string, args ...interface{}) (sql.Result, error) {
					expectedFnCall["execContext"].called++
					x := "UPDATE inventories SET stock_count = stock_count - $1 WHERE id = $2 AND stock_count >= $3"
					if q == x && expectedFnCall["execContext"].called == 3 {
						return mockSQLResult{
							rowsAffected: func() (int64, error) {
								return 0, nil
							},
						}, nil
					}

					return mockSQLResult{
						rowsAffected: func() (int64, error) {
							return 1, nil
						},
					}, nil
				},
			}, nil
		},
	}

	err := r.AdjustInventories(nil, testOrder)
	if ev, ok := err.(repositories.InventoryQuantityUpdateError); !ok {
		tt.Errorf("expecting error returned with repositories.InventoryQuantityUpdateError type, got %T", err)
	} else if ev.ProductID() != testOrder[1].ProductID {
		tt.Error("expecting error returned with correct ProductID")
	}

	for k, v := range expectedFnCall {
		if v.called != v.expecting {
			tt.Errorf("expecting calling %s only %d time, got %d", k, v.expecting, v.called)
		}
	}
}

type mockTx struct {
	commit      func() error
	rollback    func() error
	execContext func(context.Context, string, ...interface{}) (sql.Result, error)
}

func (m mockTx) Commit() error {
	return m.commit()
}

func (m mockTx) Rollback() error {
	return m.rollback()
}

func (m mockTx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return m.execContext(ctx, query, args)
}

type mockSQLResult struct {
	lastInsertID func() (int64, error)
	rowsAffected func() (int64, error)
}

func (r mockSQLResult) LastInsertId() (int64, error) {
	return r.lastInsertID()
}

func (r mockSQLResult) RowsAffected() (int64, error) {
	return r.rowsAffected()
}
