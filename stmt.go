package mysql

// Stmt 代表一个准备好的语句，可以绑定参数并执行。
type Stmt struct {
	db    db
	query string
}

type db interface {
	Exec(query string, args ...interface{}) (Result, error)
	Prepare(query string) (*Stmt, error)
	Query(query string, args ...interface{}) (*Rows, error)
	QueryRow(query string, args ...interface{}) (*Row, error)
}

// Close 关闭这条语句并释放资源。
func (s *Stmt) Close() error {
	// 由于 Go 标准库的问题，这里并没有真正使用 Stmt，也没有建立连接，所以什么都不用做。
	return nil
}

// Exec 执行一条修改语句并返回结果。
func (s *Stmt) Exec(args ...interface{}) (result Result, err error) {
	return s.db.Exec(s.query, args...)
}

// Query 查询一个带参数的查询，返回所有的结果。
func (s *Stmt) Query(args ...interface{}) (rows *Rows, err error) {
	return s.db.Query(s.query, args...)
}

// QueryRow 查询一个带参数的查询，返回第一条结果。
// 如果查询出现错误，QueryRow 依然会保证返回一个合法的 row，但是调用 row.Scan() 会报错。
func (s *Stmt) QueryRow(args ...interface{}) (row *Row, err error) {
	return s.db.QueryRow(s.query, args...)
}
