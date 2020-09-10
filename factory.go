package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/altstory/go-log"
	"github.com/altstory/go-mysql/internal/driver"
	"github.com/altstory/go-runner"
	"github.com/go-sql-driver/mysql"
)

var (
	defaultFactory = Register("mysql")
)

// Factory 代表一个用于创建 MySQL 连接的工厂。
// 创建 Factory 之后必须调用 `Factory#Conn` 方法建立连接，
// 否则后续无法通过 `Factory#New` 方法创建 MySQL 实例。
type Factory struct {
	unavailable bool // 用来标记 Factory 是否完全不可用，方便 Register 能安全的工作。

	dsn       string
	dsnSlave  string
	mod       int64
	instances []ConfigInstance

	connMaxLifeTime time.Duration
	maxIdleConns    int
	maxOpenConns    int

	connPtr unsafe.Pointer
}

// NewFactory 实例化一个工厂。
// 创建工厂后需要调用 `Factory#Conn` 才能真正建立连接，后续才能使用 `Factory#New`。
func NewFactory(config *Config) *Factory {
	if config.ConnMaxLifetime == 0 {
		config.ConnMaxLifetime = DefaultConnMaxLifetime
	}

	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = DefaultMaxIdleConns
	}

	return &Factory{
		dsn:       config.DSN, // 这里不检查合法性，等到 Conn 的时候自然知道有没有问题。
		dsnSlave:  config.DSNSlave,
		mod:       config.Mod,
		instances: config.Instances,

		connMaxLifeTime: config.ConnMaxLifetime,
		maxIdleConns:    config.MaxIdleConns,
		maxOpenConns:    config.MaxOpenConns,
	}
}

// Conn 建立 MySQL 连接。
func (f *Factory) Conn(ctx context.Context) (err error) {
	if f.unavailable {
		return errors.New("go-mysql: factory is not initialized")
	}

	// 先检查配置的合法性。
	// 如果设置了 instances，那么就得设置合法的 mod，并且 buckets 需要能覆盖 mod 所有情况、
	if err = f.validateInstances(); err != nil {
		return
	}

	conn := &dbConn{
		Instances: make(map[int64]*dbInstance),
	}

	if f.dsn != "" {
		err = conn.openDBConn(ctx, f, f.dsn, f.dsnSlave)

		if err != nil {
			return
		}
	}

	for _, ins := range f.instances {
		db := &dbInstance{}
		err = db.openDBConn(ctx, f, ins.DSN, ins.DSNSlave)

		if err != nil {
			return
		}

		for _, b := range ins.Buckets {
			conn.Instances[b] = db
		}
	}

	old := (*dbConn)(atomic.SwapPointer(&f.connPtr, unsafe.Pointer(conn)))

	if old != nil {
		old.Close()
	}

	return nil
}

func (f *Factory) validateInstances() error {
	if len(f.instances) == 0 {
		return nil
	}

	if f.mod <= 0 {
		return errors.New("go-mysql: mod should not be 0 when instances are set")
	}

	buckets := map[int64]struct{}{}

	for _, ins := range f.instances {
		for _, b := range ins.Buckets {
			if b >= f.mod {
				return fmt.Errorf("go-mysql: invalid bucket index %v which is larger than mod %v", b, f.mod)
			}

			if _, ok := buckets[b]; ok {
				return fmt.Errorf("go-mysql: bucket index %v is defined more than once", b)
			}

			buckets[b] = struct{}{}
		}
	}

	if int64(len(buckets)) != f.mod {
		missing := []int64{}

		for i := int64(0); i < f.mod; i++ {
			if _, ok := buckets[i]; !ok {
				missing = append(missing, i)
			}
		}

		return fmt.Errorf("go-mysql: indice of buckets in instances are missing %v", missing)
	}

	return nil
}

func (f *Factory) openDB(ctx context.Context, dsn string) (db *sql.DB, err error) {
	// 检查 DSN 是否合法。
	cfg, err := mysql.ParseDSN(dsn)

	if err != nil {
		log.Errorf(ctx, "err=%v||dsn=%v||go-mysql: MySQL dsn is invalid", err, dsn)
		return
	}

	// 为了支持解析 DATETIME 类型到 time.Time，设置这个标记。
	cfg.ParseTime = true
	cfg.Loc = time.Local
	db, err = sql.Open(driver.Name, cfg.FormatDSN())

	if err != nil {
		log.Errorf(ctx, "err=%v||dsn=%v||go-mysql: fail to open MySQL connection", err, dsn)
		return
	}

	db.SetConnMaxLifetime(f.connMaxLifeTime)
	db.SetMaxIdleConns(f.maxIdleConns)
	db.SetMaxOpenConns(f.maxOpenConns)

	if err = db.Ping(); err != nil {
		db = nil
		log.Errorf(ctx, "err=%v||dsn=%v||go-mysql: fail to ping MySQL", err, dsn)
		return
	}

	return
}

// New 建立新的 MySQL 实例，供业务代码使用。
func (f *Factory) New(ctx context.Context) *MySQL {
	if f.unavailable {
		panic(errors.New("go-mysql: factory is not initialized"))
	}

	conn := f.conn()

	if conn == nil {
		log.Errorf(ctx, "dsn=%v||go-mysql: MySQL factory is not connected (forgot to call `f.Conn`?)", f.dsn)
		panic(errors.New("go-mysql: factory is not connected"))
	}

	idx, ok := indexFromContext(ctx)

	if !ok || len(conn.Instances) == 0 {
		if conn.Master == nil {
			log.Errorf(ctx, "dsn=%v||instances=%v||go-mysql: no default master DSN for MySQL factory", f.dsn, f.instances)

			if len(conn.Instances) > 0 {
				panic(errors.New("go-mysql: missing instance index (forgot to call WithIndex?)"))
			} else {
				panic(errors.New("go-mysql: no default master DSN"))
			}
		}

		return newMySQL(ctx, conn.Master, conn.Slave)
	}

	if len(conn.Instances) == 0 {
		log.Errorf(ctx, "dsn=%v||instances=%v||go-mysql: no cluster instance is connected", f.dsn, f.instances)
		panic(errors.New("go-mysql: no cluster instance nor default master DSN"))
	}

	idx = idx % f.mod
	ins := conn.Instances[idx]
	return newMySQL(ctx, ins.Master, ins.Slave)
}

// Close 关闭数据库连接，一般没有调用的必要。
func (f *Factory) Close() error {
	if f.unavailable {
		return errors.New("go-mysql: factory is not initialized")
	}

	conn := (*dbConn)(atomic.SwapPointer(&f.connPtr, nil))

	if conn == nil {
		return nil
	}

	return conn.Close()
}

func (f *Factory) conn() *dbConn {
	return (*dbConn)(atomic.LoadPointer(&f.connPtr))
}

// Register 将配置文件里 [section] 部分的配置用于初始化 MySQL。
// 需要注意，Register 函数依赖于 runner 的启动流程，
// 在 AddClient 周期结束前，返回的 Factory 并不可用。
func Register(section string) **Factory {
	factory := &Factory{
		unavailable: true,
	}

	runner.AddClient(section, func(ctx context.Context, config *Config) error {
		if config == nil {
			return fmt.Errorf("go-mysql: missing MySQL config `[%v]`", section)
		}

		f := NewFactory(config)

		if err := f.Conn(ctx); err != nil {
			log.Errorf(ctx, "err=%v||dsn=%v||section=%v||go-mysql: fail to init MySQL", err, config.DSN, section)
			return err
		}

		log.Tracef(ctx, "dsn=%v||section=%v||go-mysql: mysql is connected", config.DSN, section)
		initMetrics()
		factory = f
		return nil
	})
	return &factory
}

type dbConn struct {
	dbInstance
	Instances map[int64]*dbInstance
}

type dbInstance struct {
	Master *sql.DB
	Slave  *sql.DB
}

func (conn *dbConn) Close() error {
	err := conn.dbInstance.Close()

	if err != nil {
		return err
	}

	for _, ins := range conn.Instances {
		err = ins.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

func (db *dbInstance) openDBConn(ctx context.Context, f *Factory, dsn, dsnSlave string) (err error) {
	db.Master, err = f.openDB(ctx, dsn)

	if err != nil {
		return
	}

	if dsnSlave == "" {
		db.Slave = db.Master
	} else {
		db.Slave, err = f.openDB(ctx, dsnSlave)

		if err != nil {
			return
		}
	}

	return
}

func (db *dbInstance) Close() error {
	err := db.Master.Close()

	if err != nil {
		return err
	}

	if db.Master != db.Slave {
		err = db.Slave.Close()

		if err != nil {
			return err
		}
	}

	return nil
}
