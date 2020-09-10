package mysql

import (
	"context"
	"sync"
	"time"

	"github.com/altstory/go-log"
	"github.com/altstory/go-metrics"
	"github.com/altstory/go-runner"
)

const (
	mysqlReadStatsKey         = "mysql_read"
	mysqlWriteStatsKey        = "mysql_write"
	mysqlAffectedRowsStatsKey = "mysql_affected_rows"
	mysqlSelectedRowsStatsKey = "mysql_selected_rows"
)

var mysqlMetrics struct {
	Read, Write, AffectedRows, SelectedRows *metrics.Metric
}

var metricsOnce sync.Once

func initMetrics() {
	metricsOnce.Do(func() {
		mysqlMetrics.Read = metrics.Define(&metrics.Def{
			Category: mysqlReadStatsKey,
			Method:   metrics.Sum,
		})
		mysqlMetrics.Write = metrics.Define(&metrics.Def{
			Category: mysqlWriteStatsKey,
			Method:   metrics.Sum,
		})
		mysqlMetrics.AffectedRows = metrics.Define(&metrics.Def{
			Category: mysqlAffectedRowsStatsKey,
			Method:   metrics.Sum,
		})
		mysqlMetrics.SelectedRows = metrics.Define(&metrics.Def{
			Category: mysqlSelectedRowsStatsKey,
			Method:   metrics.Sum,
		})
	})
}

func statsForRead(ctx context.Context, query string, start time.Time) {
	runner.StatsFromContext(ctx).Add(mysqlReadStatsKey, 1)
	mysqlMetrics.Read.Add(1)
	log.Tracef(ctx, "query=%v||proctime=%.6f||go-mysql: query rows", query, time.Now().Sub(start).Seconds())
}

func statsForWrite(ctx context.Context, query string, start time.Time) {
	runner.StatsFromContext(ctx).Add(mysqlWriteStatsKey, 1)
	mysqlMetrics.Write.Add(1)
	log.Tracef(ctx, "query=%v||proctime=%.6f||go-mysql: execute query", query, time.Now().Sub(start).Seconds())
}

func statsForAffectedRows(ctx context.Context, value int64) {
	runner.StatsFromContext(ctx).Add(mysqlAffectedRowsStatsKey, int(value))
	mysqlMetrics.AffectedRows.Add(value)
}

func statsForSelectedRows(ctx context.Context, value int64) {
	runner.StatsFromContext(ctx).Add(mysqlSelectedRowsStatsKey, int(value))
	mysqlMetrics.SelectedRows.Add(value)
}
