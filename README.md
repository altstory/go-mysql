# go-mysql：MySQL 客户端 #

`go-mysql` 封装了 `database/sql` 的接口，所有接口与标准库功能保持一致，修复了 Go 标准库设计和实现层面的 bug。

## 使用方法 ##

如果使用 `go-runner` 启动服务，`go-mysql` 会被自动创建，无需操心。所有配置放在配置文件的 `[mysql]` 条目下。

配置需要填写以下信息：

```ini
[mysql]
dsn = "username:password@protocol(address)/dbname?param=value"
```

业务代码需要使用 MySQL 时，直接使用 `New` 方法即可。

```go
import "git.altstory.com/altstory-framework/go-mysql"

func Foo(ctx context.Context, req *FooRequest) (res *FooResponse, err error) {
    // 这里省略各种参数检查……

    db := mysql.New(ctx)
    rows, err := db.Query("SELECT uid FROM foo WHERE status = ?", status)

    // 判断错误，使用 rows。具体代码省略……
}
```

## SQL builder 和 ORM ##

原则上不推荐使用任何 ORM，比如 [xorm](https://github.com/go-xorm/xorm)、[gorm](https://gorm.io/) 等，这些 ORM 副作用比较难以控制，且无法很好的根据 ctx 控制执行时间。

推荐使用 SQL builder 库来拼接 SQL，提升可控性并减少人工拼接的过程。推荐的库是 [go-sqlbuilder](https://github.com/huandu/go-sqlbuilder)。

## 高级用法 ##

### 主从分离 ###

配置里可以设置 `dsn_slave` 来指定一个从库，所有的读流量会走到从库，而写流量走主库。当这个配置没有设置时，所有流量都会走 `dsn` 指定的主库。

```ini
[mysql]
dsn = "username:password@protocol(address)/dbname?param=value"
dsn_slave = "username:password@protocol(address)/dbname?param=value"
```

默认情况下，所有的 `MySQL#Query`/`MySQL#QueryRow`/`Stmt#Query`/`Stmt#QueryRow` 会走从库，所有的 `Tx`/`MySQL#Exec`/`Stmt#Exec` 会走主库。

如果需要强行走主库，可以使用以下代码来指定走主库。

```go
row, err := mysql.UseMaster().QueryRow(sql, args)
```

### 在服务中使用多个 MySQL 连接 ###

在某些场景下，仅使用一个 MySQL 并不足够，那么我们可以自行构建 `Factory` 来连接更多的 MySQL 服务。

首先在配置文件中写一个新的 MySQL 连接配置。

```ini
[mysql_another]
dsn = "username:password@protocol(address)/dbname?param=value"
```

然后实例化一个新的工厂。

```go
// anotherMySQLFactory 的类型是 **Factory，是一个指针的指针。
var anotherMySQLFactory = mysql.Register("mysql_another")
```

接着，使用这个全局变量 `anotherMySQLFactory` 来创建新的 MySQL client，在业务中使用。

```go
factory := *anotherMySQLFactory
mysql := factory.New(ctx)

// 使用 mysql client 进行各种操作……
```

### 使用 MySQL 多实例集群 ###

为了能够方便的进行 MySQL 扩容，`go-mysql` 支持配置多实例，从而让未来的 MySQL 扩容变得相对简单。

例如，在上线时我们先仅设置一个数据库信息。

```ini
[mysql]
mod = 1

    [[mysql.instances]]
    dsn = "username:password@protocol(address1)/dbname?param=value"
    buckets = [0]
```

如果发现这个库扛不住，可以添加一个新的实例。这里需要注意，`mod` 直接设置成了 `10`，这是为了方便未来在扩容的时候保持实例数据的稳定。

```ini
[mysql]
mod = 10

    [[mysql.instances]]
    dsn = "username:password@protocol(address1)/dbname?param=value"
    buckets = [0, 1, 2, 3, 4]

    [[mysql.instances]]
    dsn = "username:password@protocol(address2)/dbname?param=value"
    buckets = [5, 6, 7, 8, 9]
```

又过了一段时间，我们发现 `address2` 有些问题，需要分出更多的实例，可以直接通过修改 `buckets` 来分摊 `address2` 流量。

```ini
[mysql]
mod = 10

    [[mysql.instances]]
    dsn = "username:password@protocol(address1)/dbname?param=value"
    buckets = [0, 1, 2, 3, 4]

    [[mysql.instances]]
    dsn = "username:password@protocol(address2)/dbname?param=value"
    buckets = [5, 6, 7]

    [[mysql.instances]]
    dsn = "username:password@protocol(address3)/dbname?param=value"
    buckets = [8, 9]
```

由于 MySQL 分流后，数据写入会分到不同的主库，如果我们需要再次合并这些数据库实例，则需要花费较长的时间合并数据到一个实例才行。这需要在运维的时候注意这个细节。

使用了集群配置之后，MySQL 的使用方法会发生变化，必须使用 `mysql.WithIndex` 来修饰 `ctx` 来选择实例。

```go
// 假设我们拿到了一个数据库 hash id，需要将它转化成一个 int64 类型的数据并设置到 ctx 里面去。
// 如果 id 是 int64 类型，直接设置即可；
// 如果 id 是 string，推荐使用 hash/fnv 的 fnv.New64() 来获得一个 hash 来计算，这里做了个演示。
uuid := "xxxxxxxx-xxxxx-xxxxxxxx"
hash := fnv.New64()
io.WriteString(hash, uuid)
idx := hash.Sum64()

ctx = mysql.WithIndex(ctx, idx)
m := mysql.New(ctx)
```
