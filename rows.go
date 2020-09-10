package mysql

import (
	"context"
	"database/sql"
)

// Rows 代表一个查询结果。
type Rows struct {
	ctx  context.Context
	rows *sql.Rows
}

// Close 关闭 rs 来释放资源。
func (rs *Rows) Close() error {
	return rs.rows.Close()
}

// ColumnTypes 返回列类型信息。
func (rs *Rows) ColumnTypes() ([]*sql.ColumnType, error) {
	return rs.rows.ColumnTypes()
}

// Columns 返回所有列名。
func (rs *Rows) Columns() ([]string, error) {
	return rs.rows.Columns()
}

// Err 返回当前的错误。
func (rs *Rows) Err() error {
	return rs.rows.Err()
}

// Next 查询下一条结果，如果已经没有更多结果或者出错，返回 false。
func (rs *Rows) Next() bool {
	exists := rs.rows.Next()

	if exists {
		statsForSelectedRows(rs.ctx, 1)
	}

	return exists
}

// NextResultSet 查询是否存在下一条记录，但是并不会真的返回下一条结果。
// 真正 Scan 之前，还得调用 Next 来实际获取这条结果。
func (rs *Rows) NextResultSet() bool {
	return rs.rows.NextResultSet()
}

// Scan 将查询出来的数据设置到 dest 里面。
func (rs *Rows) Scan(dest ...interface{}) error {
	return rs.rows.Scan(dest...)
}
