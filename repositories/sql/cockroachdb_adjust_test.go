package sql

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"tomshop/repositories"
)

func TestCockroachRepo_AdjustInventories(t *testing.T) {
	t.Run("must return error if cannot start new Txn", errorWhenStartTxn)
	t.Run("must return error if cannot create new prepared-statement", errorWhenPrepareStatement)
	t.Run("must return nil error if input are empty", inputEmptyNilError)
	t.Run("must Exec statement with correct args", execContextWithCorrectArgs)
	t.Run("must Rollback if any error happen when calling Executor.ExecContext", errorInExecContext)
	t.Run("must Rollback even if panic in execContext", panicInExecContext)
	t.Run("must Rollback when cannot adjust any item", rollBackWhenNoRowUpdated)
	t.Run("must return error if cannot commit", errorWhenCommitError)
}

var testInventories = []repositories.Inventory{
	{
		ProductID:  1,
		StockCount: 11,
		Version:    1111,
	},
	{
		ProductID:  2,
		StockCount: 22,
		Version:    222,
	},
}

func errorWhenStartTxn(tt *testing.T) {
	dummyErr := fmt.Errorf("dummy error")
	r := &CockroachRepo{
		txnFactory: func(opts *sql.TxOptions) (TransactionManager, error) {
			return nil, dummyErr
		},
	}

	if !reflect.DeepEqual(r.AdjustInventories(testInventories), dummyErr) {
		tt.Error("expecting error return if txnFactory cannot create new transaction")
	}
}

func errorWhenPrepareStatement(tt *testing.T) {
	dummyErr := fmt.Errorf("dummy stmtFactory error")
	r := &CockroachRepo{
		txnFactory: func(opts *sql.TxOptions) (TransactionManager, error) {
			return &sql.Tx{}, nil
		},
		stmtFactory: func(tm TransactionManager, query string) (Executor, error) {
			return nil, dummyErr
		},
	}
	err := r.AdjustInventories(testInventories)
	if !reflect.DeepEqual(err, dummyErr) {
		tt.Log(err)
		tt.Error("expecting error return if txnFactory cannot create new prepared-statement")
	}
}

func inputEmptyNilError(tt *testing.T) {
	r := &CockroachRepo{
		txnFactory: func(opts *sql.TxOptions) (TransactionManager, error) {
			return &sql.Tx{}, nil
		},
		stmtFactory: func(tm TransactionManager, query string) (Executor, error) {
			return &sql.Stmt{}, nil
		},
	}

	if err := r.AdjustInventories(nil); err != nil {
		tt.Log(err)
		tt.Error("expecting error return nil error", err)

	}
}

func execContextWithCorrectArgs(tt *testing.T) {
	expectingExecCallArgs := make(map[int64][]interface{}, 2)
	expectingCommitCalled := 0
	expectingCloseCalled := 0
	r := &CockroachRepo{
		ctx: context.Background(),
		txnFactory: func(opts *sql.TxOptions) (TransactionManager, error) {
			return mockTransactionManager{
				commit: func() error {
					expectingCommitCalled++
					return nil
				},
			}, nil
		},
		stmtFactory: func(tm TransactionManager, query string) (Executor, error) {
			return mockExecutor{
				execContext: func(c context.Context, args ...interface{}) (sql.Result, error) {
					if len(args) > 3 {
						tt.Fatal("ExecContext expecting only 3 args when, got", args)
					}
					expectingExecCallArgs[args[0].(int64)] = args
					return mockSQLResult{
						rowsAffected: func() (int64, error) {
							return 1, nil
						},
					}, nil
				},
				close: func() error {
					expectingCloseCalled++
					return nil
				},
			}, nil
		},
	}

	if err := r.AdjustInventories(testInventories); err != nil {
		tt.Error("unexpected error", err)
	}

	if expectingCommitCalled != 1 {
		tt.Error("expected TransactionManager.Commit called only 1, got", expectingCommitCalled)
	}

	if expectingCloseCalled != 1 {
		tt.Error("expected Executor.Close called only 1 time, got", expectingCloseCalled)
	}

	if len(expectingExecCallArgs) != len(testInventories) {
		tt.Errorf("expected Executor.ExecContext called %d times, got %d", len(testInventories), len(expectingExecCallArgs))
	}

	for _, inventory := range testInventories {
		args := expectingExecCallArgs[inventory.ProductID]
		var a, b, c int64
		a = args[0].(int64)
		b = args[1].(int64)
		c = args[2].(int64)

		if a != inventory.ProductID || b != inventory.StockCount || c != inventory.Version {
			tt.Errorf(
				"expected Executor.ExecContext called with correct order of %d %d %d",
				inventory.ProductID,
				inventory.StockCount,
				inventory.Version,
			)
		}
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
		"close": &expectingCall{
			expecting: 1,
		},
		"execContext": &expectingCall{
			expecting: 1,
		},
	}

	r := &CockroachRepo{
		ctx: context.Background(),
		txnFactory: func(opts *sql.TxOptions) (TransactionManager, error) {
			return mockTransactionManager{
				commit: func() error {
					expectedFnCall["commit"].called++
					return nil
				},
				rollback: func() error {
					expectedFnCall["rollback"].called++
					return nil
				},
			}, nil
		},
		stmtFactory: func(tm TransactionManager, query string) (Executor, error) {
			return mockExecutor{
				execContext: func(c context.Context, args ...interface{}) (sql.Result, error) {
					expectedFnCall["execContext"].called++
					return nil, dummyErr
				},
				close: func() error {
					expectedFnCall["close"].called++
					return nil
				},
			}, nil
		},
	}

	if err := r.AdjustInventories(testInventories); !reflect.DeepEqual(err, dummyErr) {
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
		"close": &expectingCall{
			expecting: 1,
		},
		"execContext": &expectingCall{
			expecting: 1,
		},
	}

	r := &CockroachRepo{
		ctx: context.Background(),
		txnFactory: func(opts *sql.TxOptions) (TransactionManager, error) {
			return mockTransactionManager{
				commit: func() error {
					expectedFnCall["commit"].called++
					return nil
				},
				rollback: func() error {
					expectedFnCall["rollback"].called++
					return nil
				},
			}, nil
		},
		stmtFactory: func(tm TransactionManager, query string) (Executor, error) {
			return mockExecutor{
				execContext: func(c context.Context, args ...interface{}) (sql.Result, error) {
					expectedFnCall["execContext"].called++
					panic("dummyPannic")
				},
				close: func() error {
					expectedFnCall["close"].called++
					return nil
				},
			}, nil
		},
	}

	if err := r.AdjustInventories(testInventories); !reflect.DeepEqual(err, dummyErr) {
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
		"close": &expectingCall{
			expecting: 1,
		},
		"execContext": &expectingCall{
			expecting: 2,
		},
	}

	r := &CockroachRepo{
		ctx: context.Background(),
		txnFactory: func(opts *sql.TxOptions) (TransactionManager, error) {
			return mockTransactionManager{
				commit: func() error {
					expectedFnCall["commit"].called++
					return nil
				},
				rollback: func() error {
					expectedFnCall["rollback"].called++
					return nil
				},
			}, nil
		},
		stmtFactory: func(tm TransactionManager, query string) (Executor, error) {
			return mockExecutor{
				execContext: func(c context.Context, args ...interface{}) (sql.Result, error) {
					expectedFnCall["execContext"].called++
					if expectedFnCall["execContext"].called == 1 { // pass the first call
						return mockSQLResult{
							rowsAffected: func() (int64, error) {
								return 1, nil
							},
						}, nil
					}

					return mockSQLResult{
						rowsAffected: func() (int64, error) {
							return 0, nil
						},
					}, nil
				},
				close: func() error {
					expectedFnCall["close"].called++
					return nil
				},
			}, nil
		},
	}

	err := r.AdjustInventories(testInventories)
	if ev, ok := err.(repositories.InventoryQuantityUpdateError); !ok {
		tt.Errorf("expecting error returned with repositories.InventoryQuantityUpdateError type, got %T", err)
	} else if ev.ProductID() != testInventories[1].ProductID {
		tt.Error("expecting error returned with correct ProductID")
	}

	for k, v := range expectedFnCall {
		if v.called != v.expecting {
			tt.Errorf("expecting calling %s only %d time, got %d", k, v.expecting, v.called)
		}
	}
}

func errorWhenCommitError(tt *testing.T) {
	type expectingCall struct {
		expecting int
		called    int
	}
	expectedFnCall := map[string]*expectingCall{
		"commit": &expectingCall{
			expecting: 1,
		},
		"rollback": &expectingCall{
			expecting: 0,
		},
		"close": &expectingCall{
			expecting: 1,
		},
		"execContext": &expectingCall{
			expecting: 2,
		},
	}

	dummyErr := fmt.Errorf("dummyCommitError")
	r := &CockroachRepo{
		ctx: context.Background(),
		txnFactory: func(opts *sql.TxOptions) (TransactionManager, error) {
			return mockTransactionManager{
				commit: func() error {
					expectedFnCall["commit"].called++
					return dummyErr
				},
				rollback: func() error {
					expectedFnCall["rollback"].called++
					return nil
				},
			}, nil
		},
		stmtFactory: func(tm TransactionManager, query string) (Executor, error) {
			return mockExecutor{
				execContext: func(c context.Context, args ...interface{}) (sql.Result, error) {
					expectedFnCall["execContext"].called++
					return mockSQLResult{
						rowsAffected: func() (int64, error) {
							return 1, nil
						},
					}, nil
				},
				close: func() error {
					expectedFnCall["close"].called++
					return nil
				},
			}, nil
		},
	}
	err := r.AdjustInventories(testInventories)
	for k, v := range expectedFnCall {
		if v.called != v.expecting {
			tt.Errorf("expecting calling %s only %d time, got %d", k, v.expecting, v.called)
		}
	}

	if !reflect.DeepEqual(err, dummyErr) {
		tt.Error("expected dummy commit error, got", err)
	}
}

type mockTransactionManager struct {
	commit   func() error
	rollback func() error
}

func (m mockTransactionManager) Commit() error {
	return m.commit()
}

func (m mockTransactionManager) Rollback() error {
	return m.rollback()
}

type mockExecutor struct {
	execContext func(context.Context, ...interface{}) (sql.Result, error)
	close       func() error
}

func (e mockExecutor) ExecContext(c context.Context, args ...interface{}) (sql.Result, error) {
	return e.execContext(c, args...)
}

func (e mockExecutor) Close() error {
	return e.close()
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
func TestNewCockroachRepoStmtFactory(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("stmtFactory must panic if recieved dummy TransactionManager")
		}
	}()

	r := NewCockroachRepo(context.Background(), nil)
	r.stmtFactory(TransactionManager(nil), "test")
}
