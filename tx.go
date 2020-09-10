package mysql

import (
	"context"
	"database/sql"
	"time"
)

// Tx 代表一个事务。
type Tx struct {
	ctx context.Context
	tx  *sql.Tx
}

// Commit 提交事务。
func (tx *Tx) Commit() (err error) {
	if err = tx.ctx.Err(); err != nil {
		tx.Rollback()
		return
	}

	return tx.tx.Commit()
}

// Exec 执行一条修改语句并返回结果。
func (tx *Tx) Exec(query string, args ...interface{}) (result Result, err error) {
	if err = tx.ctx.Err(); err != nil {
		tx.Rollback()
		return
	}

	start := time.Now()
	sqlresult, err := tx.tx.ExecContext(tx.ctx, query, args...)
	statsForWrite(tx.ctx, query, start)

	if err != nil {
		return
	}

	affected, _ := sqlresult.RowsAffected()

	if affected > 0 {
		statsForAffectedRows(tx.ctx, affected)
	}

	result = sqlresult.(Result)
	return
}

// Prepare 准备一个 Stmt，方便绑定参数。
func (tx *Tx) Prepare(query string) (stmt *Stmt, err error) {
	if err = tx.ctx.Err(); err != nil {
		tx.Rollback()
		return
	}

	stmt = &Stmt{
		db:    tx,
		query: query,
	}
	return
}

// Query 查询一个带参数的查询，返回所有的结果。
func (tx *Tx) Query(query string, args ...interface{}) (rows *Rows, err error) {
	if err = tx.ctx.Err(); err != nil {
		tx.Rollback()
		return
	}

	start := time.Now()
	sqlrows, err := tx.tx.QueryContext(tx.ctx, query, args...)
	statsForRead(tx.ctx, query, start)

	if err != nil {
		return
	}

	rows = &Rows{
		ctx:  tx.ctx,
		rows: sqlrows,
	}
	return
}

// QueryRow 查询一个带参数的查询，返回第一条结果。
// 如果查询出现错误，QueryRow 依然会保证返回一个合法的 row，但是调用 row.Scan() 会报错。
func (tx *Tx) QueryRow(query string, args ...interface{}) (row *Row, err error) {
	if err = tx.ctx.Err(); err != nil {
		tx.Rollback()
		return
	}

	start := time.Now()
	sqlrow := tx.tx.QueryRowContext(tx.ctx, query, args...)
	statsForRead(tx.ctx, query, start)
	row = &Row{
		ctx: tx.ctx,
		row: sqlrow,
	}
	return
}

// Rollback 回滚事务。
func (tx *Tx) Rollback() error {
	return tx.tx.Rollback()
}

// Stmt 将一个指定的 stmt 纳入到事务 tx 的管理范围内，使其受到 commit 和 rollback 的控制。
func (tx *Tx) Stmt(stmt *Stmt) (txStmt *Stmt, err error) {
	if err = tx.ctx.Err(); err != nil {
		tx.Rollback()
		return
	}

	txStmt = &Stmt{
		db:    tx,
		query: stmt.query,
	}
	return
}
