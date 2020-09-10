package mysql

import (
	"context"
	"testing"

	"github.com/huandu/go-sqlbuilder"
)

const testDB = "xxxxxxx_test"

// mysqlFactory 返回 MySQL Factory，如果这个返回 nil，意味着 MySQL 服务没启动，测试用例会被跳过。
//
// MySQL 服务需要符合以下条件才能被连接：
//     - 监听地址 127.0.0.1:3306
//     - 用户名/密码：xxxxxxx/xxxxxxx
//     - 数据库：xxxxxxx_test
//
// 每次执行测试前都会清理整个数据库，通过 DROP DATABASE xxxxxxx_test 实现，
// 一定确保数据库没有需要保留的东西。
func mysqlFactory(t *testing.T) *Factory {
	c := &Config{
		DSN: "xxxxxxxx:xxxxxxxx@tcp(127.0.0.1:3306)/" + testDB,
	}
	f := NewFactory(c)
	ctx := context.Background()

	if err := f.Conn(ctx); err != nil {
		t.Skipf("MySQL is not available. [dns:%v]", c.DSN)
		return nil
	}

	return f
}

// mysqlClusterFactory 返回 MySQL Factory，这里使用了 cluster 模式。
// 详细信息见 mysqlFactory 文档。
func mysqlClusterFactory(t *testing.T) *Factory {
	c := &Config{
		Mod: 4,
		Instances: []ConfigInstance{
			{
				DSN:     "xxxxxxxx:xxxxxxxx@tcp(127.0.0.1:3306)/" + testDB,
				Buckets: []int64{0, 1, 2, 3},
			},
		},
	}
	f := NewFactory(c)
	ctx := context.Background()

	if err := f.Conn(ctx); err != nil {
		t.Skipf("MySQL is not available. [dns:%v]", c.DSN)
		return nil
	}

	return f
}

func initTable(ctx context.Context, t *testing.T, f *Factory, table string, defs ...string) *MySQL {
	mysql := f.New(ctx)

	if _, err := mysql.Exec("DROP TABLE " + table); err != nil {
		t.Fatalf("unable to drop table. [err:%v] [table:%v]", err, table)
	}

	ctb := sqlbuilder.NewCreateTableBuilder()
	ctb.CreateTable(table).IfNotExists()

	for _, def := range defs {
		ctb.Define(def)
	}
	sql, args := ctb.Build()

	if _, err := mysql.Exec(sql, args...); err != nil {
		t.Fatalf("unable to create table. [err:%v]", err)
	}

	return mysql
}
