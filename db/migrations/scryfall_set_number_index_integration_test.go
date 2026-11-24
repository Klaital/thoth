//go:build integration

package migrations_test

import (
	"testing"

	"github.com/klaital/thoth/db/migrations"
	"github.com/klaital/thoth/internal/testdb"
)

func TestMigration011ScryfallSetNumberIndex(t *testing.T) {
	db, dsn := testdb.NewEmpty(t)
	m, closeFn, err := migrations.New(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer closeFn()

	hasIndex := func(name string) bool {
		return count(t, db, `SELECT COUNT(*) FROM information_schema.statistics
			WHERE table_schema = DATABASE() AND table_name = 'scryfall_cards' AND index_name = '`+name+`'`) > 0
	}

	if err := m.Migrate(11); err != nil {
		t.Fatalf("migrate to 11: %v", err)
	}
	if !hasIndex("idx_scryfall_cards_set_number") || hasIndex("idx_scryfall_cards_set") {
		t.Error("after up: want only the (set_code, collector_number) index")
	}

	if err := m.Steps(-1); err != nil {
		t.Fatalf("down: %v", err)
	}
	if hasIndex("idx_scryfall_cards_set_number") || !hasIndex("idx_scryfall_cards_set") {
		t.Error("after down: want only the set_code index")
	}
	if err := m.Steps(1); err != nil {
		t.Fatalf("up again: %v", err)
	}
}
