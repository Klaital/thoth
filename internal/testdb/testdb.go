// Package testdb gives integration tests a throwaway MySQL database.
//
// THOTH_TEST_DB_DSN must name a server (the database part is ignored) and a
// user allowed to CREATE and DROP databases, e.g. the compose db service:
//
//	root:thoth-root@tcp(db:3306)/?parseTime=true&interpolateParams=true
//
// Each call creates a uniquely named thoth_it_* database and drops it when the
// test ends, so tests can run in parallel and never touch real data.
package testdb

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"os"
	"testing"

	"github.com/go-sql-driver/mysql"

	"github.com/klaital/thoth/db/migrations"
)

const envDSN = "THOTH_TEST_DB_DSN"

// New returns a connection to a fresh database with all migrations applied.
func New(t testing.TB) *sql.DB {
	t.Helper()
	db, dsn := NewEmpty(t)
	if err := migrations.Up(dsn); err != nil {
		t.Fatalf("migrating test database: %v", err)
	}
	return db
}

// NewEmpty returns a connection to a fresh, unmigrated database and its DSN,
// for tests that drive migrations themselves.
func NewEmpty(t testing.TB) (*sql.DB, string) {
	t.Helper()
	raw := os.Getenv(envDSN)
	if raw == "" {
		t.Fatalf("%s is not set; run integration tests with `make test-integration`", envDSN)
	}
	cfg, err := mysql.ParseDSN(raw)
	if err != nil {
		t.Fatalf("parsing %s: %v", envDSN, err)
	}
	cfg.ParseTime = true

	admin, err := sql.Open("mysql", withDB(cfg, ""))
	if err != nil {
		t.Fatalf("connecting to test server: %v", err)
	}
	t.Cleanup(func() { admin.Close() })

	name := "thoth_it_" + randomSuffix(t)
	if _, err := admin.Exec("CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4"); err != nil {
		t.Fatalf("creating %s: %v", name, err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP DATABASE IF EXISTS `" + name + "`"); err != nil {
			t.Errorf("dropping %s: %v", name, err)
		}
	})

	dsn := withDB(cfg, name)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("connecting to %s: %v", name, err)
	}
	// Registered after the DROP cleanup, so it runs first.
	t.Cleanup(func() { db.Close() })
	return db, dsn
}

func withDB(cfg *mysql.Config, name string) string {
	c := cfg.Clone()
	c.DBName = name
	return c.FormatDSN()
}

func randomSuffix(t testing.TB) string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("random suffix: %v", err)
	}
	return hex.EncodeToString(b)
}
