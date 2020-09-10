package driver

import (
	"database/sql"

	"github.com/go-sql-driver/mysql"
)

// Name 是注册的 MySQL driver 名字。
const Name = "altstory-mysql"

func init() {
	// 当前只是简单的使用了开源的 driver，后续要实现测试用内存数据库时，会改为一个自行实现的版本。
	sql.Register(Name, &mysql.MySQLDriver{})
}
