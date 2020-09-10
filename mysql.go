package mysql

import (
	"context"
	"database/sql"
	"time"

	"github.com/altstory/go-log"
)

// MySQL 代表一个数据库的链接。
type MySQL struct {
	ctx       context.Context
	master    *sql.DB
	slave     *sql.DB
	useMaster bool
}

// New 通过默认工厂创建一个 MySQL 实例。
func New(ctx context.Context) *MySQL {
	factory := *defaultFactory

	if factory == nil {
		log.Errorf(ctx, "go-mysql: default factory is not initialized (forgot to set MySQL config?)")
		return nil
	}

	return factory.New(ctx)
}

func newMySQL(ctx context.Context, master, slave *sql.DB) *MySQL {
	return &MySQL{
		ctx:    ctx,
		master: master,
		slave:  slave,
	}
}

// BeginTx 开始一个事务。
func (mysql *MySQL) BeginTx(opts *sql.TxOptions) (tx *Tx, err error) {
	if err = mysql.ctx.Err(); err != nil {
		return
	}

	sqltx, err := mysql.db(true).BeginTx(mysql.ctx, opts)

	if err != nil {
		return
	}

	tx = &Tx{
		ctx: mysql.ctx,
		tx:  sqltx,
	}
	return
}

// Exec 执行一条修改语句并返回结果。
func (mysql *MySQL) Exec(query string, args ...interface{}) (result Result, err error) {
	if err = mysql.ctx.Err(); err != nil {
		return
	}

	start := time.Now()
	res, err := mysql.db(true).ExecContext(mysql.ctx, query, args...)
	statsForWrite(mysql.ctx, query, start)

	if err != nil {
		return
	}

	affected, _ := res.RowsAffected()

	if affected > 0 {
		statsForAffectedRows(mysql.ctx, affected)
	}

	result = res.(Result)
	return
}

// Ping 测试连接是否可用。
func (mysql *MySQL) Ping() (err error) {
	if err = mysql.ctx.Err(); err != nil {
		return
	}

	return mysql.db(false).PingContext(mysql.ctx)
}

// Prepare 准备一个 Stmt，方便绑定参数。
func (mysql *MySQL) Prepare(query string) (stmt *Stmt, err error) {
	if err = mysql.ctx.Err(); err != nil {
		return
	}

	stmt = &Stmt{
		db:    mysql,
		query: query,
	}
	return
}

// Query 查询一个带参数的查询，返回所有的结果。
func (mysql *MySQL) Query(query string, args ...interface{}) (rows *Rows, err error) {
	if err = mysql.ctx.Err(); err != nil {
		return
	}

	start := time.Now()
	sqlrows, err := mysql.db(false).QueryContext(mysql.ctx, query, args...)
	statsForRead(mysql.ctx, query, start)

	if err != nil {
		return
	}

	rows = &Rows{
		ctx:  mysql.ctx,
		rows: sqlrows,
	}
	return
}

// QueryRow 查询一个带参数的查询，返回第一条结果。
// 如果查询出现错误，QueryRow 依然会保证返回一个合法的 row，但是调用 row.Scan() 会报错。
func (mysql *MySQL) QueryRow(query string, args ...interface{}) (row *Row, err error) {
	if err = mysql.ctx.Err(); err != nil {
		return
	}

	start := time.Now()
	sqlrow := mysql.db(false).QueryRowContext(mysql.ctx, query, args...)
	statsForRead(mysql.ctx, query, start)
	row = &Row{
		ctx: mysql.ctx,
		row: sqlrow,
	}
	return
}

// Stats 返回数据库当前状态。
func (mysql *MySQL) Stats() sql.DBStats {
	return mysql.db(false).Stats()
}

// UseMaster 返回一个 MySQL 实例，调用这个实例的所有方法都会调用主库。
func (mysql *MySQL) UseMaster() *MySQL {
	cp := *mysql
	cp.useMaster = true
	return &cp
}

func (mysql *MySQL) db(forceMaster bool) *sql.DB {
	if mysql.useMaster || forceMaster {
		return mysql.master
	}

	return mysql.slave
}
