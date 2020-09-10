package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/huandu/go-assert"
	"github.com/huandu/go-sqlbuilder"
)

func TestMySQLConnFailed(t *testing.T) {
	f := NewFactory(&Config{
		DSN: "invalid",
	})
	ctx := context.Background()

	if err := f.Conn(ctx); err == nil {
		t.Fatalf("f.Conn should fail.")
	}
}

type testSimpleCommand struct {
	ID        int64     `db:"id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
}

func TestMySQLSimpleCommand(t *testing.T) {
	a := assert.New(t)
	const table = "test_simple_command"
	tm, _ := time.Parse(time.RFC3339, "2019-07-11T12:34:56+08:00")
	name := "bbbb"

	runFunc := func(f *Factory) {
		ctx := WithIndex(context.Background(), 1234)
		mysql := initTable(ctx, t, f, table,
			"id bigint(20) NOT NULL",
			"name VARCHAR(255) NOT NULL",
			"created_at DATETIME NOT NULL",
		)

		ib := sqlbuilder.NewInsertBuilder()
		ib.InsertInto(table)
		ib.Cols("id", "name", "created_at")
		ib.Values(1, "aaaaa", time.Now())
		ib.Values(2, name, tm)
		sql, args := ib.Build()

		a.NilError(mysql.Exec(sql, args...))

		st := sqlbuilder.NewStruct(new(testSimpleCommand))
		sb := st.SelectFrom(table)
		sb.Where(sb.E("name", name))
		sb.Limit(1)
		sql, args = sb.Build()
		var simple testSimpleCommand
		row, err := mysql.QueryRow(sql, args...)

		a.NilError(err)
		a.NilError(row.Scan(st.Addr(&simple)...))
		a.Assert(simple.CreatedAt.Equal(tm))
	}

	runFunc(mysqlFactory(t))
	runFunc(mysqlClusterFactory(t))
}
