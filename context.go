package mysql

import "context"

type mysqlIndex struct{}

var keyMySQLIndex mysqlIndex

// WithIndex 在 ctx 中设置 idx，用来选择使用哪个 MySQL 实例。
func WithIndex(ctx context.Context, idx int64) context.Context {
	return context.WithValue(ctx, keyMySQLIndex, idx)
}

func indexFromContext(ctx context.Context) (idx int64, ok bool) {
	v := ctx.Value(keyMySQLIndex)

	if v == nil {
		return
	}

	idx = v.(int64)
	ok = true
	return
}
