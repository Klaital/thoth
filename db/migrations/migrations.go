package migrations

import (
	"database/sql"
	"embed"
	"errors"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	migmysql "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.sql
var FS embed.FS

// New returns a migrator for the database in dsn. It opens its own
// connection with multiStatements=true, which migration files with more than
// one statement need; call the returned close func when done.
func New(dsn string) (*migrate.Migrate, func(), error) {
	if strings.Contains(dsn, "?") {
		dsn += "&multiStatements=true"
	} else {
		dsn += "?multiStatements=true"
	}
	migDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, nil, err
	}
	src, err := iofs.New(FS, ".")
	if err != nil {
		migDB.Close()
		return nil, nil, err
	}
	driver, err := migmysql.WithInstance(migDB, &migmysql.Config{})
	if err != nil {
		migDB.Close()
		return nil, nil, err
	}
	m, err := migrate.NewWithInstance("iofs", src, "mysql", driver)
	if err != nil {
		migDB.Close()
		return nil, nil, err
	}
	return m, func() { migDB.Close() }, nil
}

// Up applies all pending migrations.
func Up(dsn string) error {
	m, closeFn, err := New(dsn)
	if err != nil {
		return err
	}
	defer closeFn()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
