package sql

import (
	"context"
	"testing"
)

func TestCockroachRepo_AdjustInventories(t *testing.T) {

}

func TestNewCockroachRepoStmtFactory(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("stmtFactory if recieved dummy TransactionManager")
		}
	}()

	r := NewCockroachRepo(context.Background(), nil)
	r.stmtFactory(TransactionManager(nil), "test")
}
