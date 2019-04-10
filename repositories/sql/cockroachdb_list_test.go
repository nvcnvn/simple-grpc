package sql

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"

	"github.com/lib/pq"
)

func TestCockroachRepo_ListInventories(t *testing.T) {
	ids := []int64{1, 2, 3}
	r := &CockroachRepo{
		querier: mockQuerier{
			t:              t,
			expectingQuery: "SELECT id, stock_count, version FROM inventories WHERE id IN ($1)",
			expectingArgs: []interface{}{
				pq.Array(ids),
			},
		},
	}

	result, err := r.ListInventories(context.Background(), ids)
	if result != nil {
		t.Error("expecting nil result, got", r)
	}

	if err.Error() != "dummyError" {
		t.Error("expecting dummyError, got", err)
	}
}

type mockQuerier struct {
	t              *testing.T
	expectingQuery string
	expectingArgs  []interface{}
}

func (m mockQuerier) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if m.expectingQuery != query {
		m.t.Errorf("expecting query %s, got %s", m.expectingQuery, query)
	}

	if !reflect.DeepEqual(m.expectingArgs, args) {
		m.t.Error("unexpected args recieved, got", args)
	}

	return nil, fmt.Errorf("dummyError")
}
