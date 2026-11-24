package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"time"

	"github.com/XSAM/otelsql"
	_ "github.com/go-sql-driver/mysql"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

// Pool tuning. database/sql transparently replaces dead connections, but only
// once it discovers they're dead — typically by failing a query. Recycling
// connections well before MySQL's wait_timeout (or any NAT/firewall idle cutoff)
// keeps the pool from handing out connections the server has already dropped.
const (
	connMaxLifetime = 5 * time.Minute
	connMaxIdleTime = 2 * time.Minute
	maxOpenConns    = 25
	maxIdleConns    = 10

	connectAttempts = 10
	maxBackoff      = 30 * time.Second
)

// Connect opens a MySQL connection pool configured to survive dropped
// connections, retrying the initial ping with exponential backoff so the
// process tolerates the database starting late or restarting.
func Connect(ctx context.Context, dsn string) (*sql.DB, error) {
	// otelsql traces each query as a child of the caller's span and records
	// query latency; both are no-ops unless telemetry is enabled.
	sqlDB, err := otelsql.Open("mysql", dsn,
		otelsql.WithAttributes(semconv.DBSystemNameMySQL),
		otelsql.WithSpanOptions(otelsql.SpanOptions{
			OmitRows:             true, // one span per row scan is noise
			OmitConnResetSession: true,
			DisableErrSkip:       true,
			// Only trace queries made inside a request. Background work
			// (migrations, usage gauges, the Scryfall sync's ~100k upserts)
			// would otherwise produce floods of orphan root spans.
			SpanFilter: func(ctx context.Context, _ otelsql.Method, _ string, _ []driver.NamedValue) bool {
				return trace.SpanContextFromContext(ctx).IsValid()
			},
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("opening DB: %w", err)
	}
	if _, err := otelsql.RegisterDBStatsMetrics(sqlDB, otelsql.WithAttributes(semconv.DBSystemNameMySQL)); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("registering DB pool metrics: %w", err)
	}
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)

	backoff := time.Second
	for attempt := 1; ; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = sqlDB.PingContext(pingCtx)
		cancel()
		if err == nil {
			return sqlDB, nil
		}
		if attempt >= connectAttempts {
			sqlDB.Close()
			return nil, fmt.Errorf("pinging DB after %d attempts: %w", attempt, err)
		}
		log.Printf("DB not reachable (attempt %d/%d): %v; retrying in %s", attempt, connectAttempts, err, backoff)
		select {
		case <-ctx.Done():
			sqlDB.Close()
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, maxBackoff)
	}
}
