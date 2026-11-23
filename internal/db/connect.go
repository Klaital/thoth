package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
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
	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening DB: %w", err)
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
