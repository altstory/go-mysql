package mysql

import (
	"context"
	"database/sql"
)

// Row 代表一条查询结果。
type Row struct {
	ctx context.Context
	row *sql.Row
}

// Scan 将查询出来的数据设置到 dest 里面。
func (r *Row) Scan(dest ...interface{}) error {
	err := r.row.Scan(dest...)

	if err != nil {
		return err
	}

	statsForSelectedRows(r.ctx, 1)
	return nil
}
