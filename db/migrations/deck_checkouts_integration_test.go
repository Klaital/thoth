//go:build integration

package migrations_test

import (
	"testing"

	"github.com/klaital/thoth/db/migrations"
	"github.com/klaital/thoth/internal/testdb"
)

func TestMigration010DeckCheckouts(t *testing.T) {
	db, dsn := testdb.NewEmpty(t)
	m, closeFn, err := migrations.New(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer closeFn()
	if err := m.Migrate(10); err != nil {
		t.Fatalf("migrate to 10: %v", err)
	}
	exec(t, db, `
INSERT INTO users (id, email) VALUES (1,'a@example.com');
INSERT INTO scryfall_cards (id,name,set_code,collector_number) VALUES ('p1','Sol Ring','c21','263');
INSERT INTO collections (id,owner_id,name,type) VALUES (10,1,'Box','mtg_cards');
INSERT INTO collection_cards (id,collection_id,scryfall_card_id,quantity) VALUES (100,10,'p1',2);
INSERT INTO decks (id,owner_id,name) VALUES (5,1,'Deck');
INSERT INTO deck_checkouts (deck_id,collection_card_id,quantity) VALUES (5,100,1);
`)
	if _, err := db.Exec(`INSERT INTO deck_checkouts (deck_id,collection_card_id,quantity) VALUES (5,100,0)`); err == nil {
		t.Error("zero-quantity checkout was allowed")
	}
	if _, err := db.Exec(`DELETE FROM collection_cards WHERE id = 100`); err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM deck_checkouts`); n != 0 {
		t.Errorf("checkouts after deleting the card = %d, want 0 (cascade)", n)
	}

	if err := m.Steps(-1); err != nil {
		t.Fatalf("down: %v", err)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'deck_checkouts'`); n != 0 {
		t.Error("deck_checkouts still exists after down")
	}
	if err := m.Steps(1); err != nil {
		t.Fatalf("up again: %v", err)
	}
}
