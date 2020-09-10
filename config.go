package mysql

import "time"

const (
	// DefaultConnMaxLifetime 代表默认的连接的最大保持时间，当前设置为 1h 时间。
	DefaultConnMaxLifetime time.Duration = time.Hour

	// DefaultMaxIdleConns 代表默认的最大空闲连接数，当前设置为 10。
	DefaultMaxIdleConns = 10
)

// Config 代表 MySQL 的配置。
type Config struct {
	DSN      string `config:"dsn"`       // DSN 是 MySQL 主库的连接字符串。
	DSNSlave string `config:"dsn_slave"` // DSNSlave 是从库的 MySQL 连接字符串，所有只读的 Query/QueryRow 都会走这个连接，默认与 DSN 相同。

	Mod       int64            `config:"mod"`       // Mod 是 hash 分桶的余数，比如设置为 10 就会将 hash%10 来计算命中哪一个实例，默认不分桶。
	Instances []ConfigInstance `config:"instances"` // Instances 是分桶后的数据库连接配置。

	ConnMaxLifetime time.Duration `config:"conn_max_life_time"` // ConnMaxLifetime 设置连接的最大保持时间，默认是 DefaultConnMaxLifetime。
	MaxIdleConns    int           `config:"max_idle_conns"`     // MaxIdleConns 设置最多保持多少个空闲连接，默认是 DefaultMaxIdleConns。
	MaxOpenConns    int           `config:"max_open_conns"`     // MaxOpenConns 设置最大同时连接数，默认是不限制。
}

// ConfigInstance 代表一组 MySQL 实例的连接字符串。
type ConfigInstance struct {
	DSN      string `config:"dsn"`       // DSN 是 MySQL 主库的连接字符串。
	DSNSlave string `config:"dsn_slave"` // DSNSlave 是从库的 MySQL 连接字符串，所有只读的 Query/QueryRow 都会走这个连接，默认与 DSN 相同。

	Buckets []int64 `config:"buckets"` // Buckets 表示这个实例对应的 bucket 号，可以是多个号，比如 [0, 1, 2]。
}
