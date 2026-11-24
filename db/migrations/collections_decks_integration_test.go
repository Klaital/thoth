//go:build integration

package migrations_test

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/klaital/thoth/db/migrations"
	"github.com/klaital/thoth/internal/testdb"
)

// Data as it looked before 000009: card and deck collections sharing
// card_items, with categories.
const seedV8 = `
INSERT INTO users (id, email) VALUES (1,'a@example.com'),(2,'b@example.com'),(3,'c@example.com');
INSERT INTO scryfall_cards (id,name,set_code,collector_number) VALUES
 ('p1','City of Brass','mh2','259'),('p2','City of Brass','5ed','412'),
 ('p3','Sol Ring','c21','263'),('p4','Atraxa, Praetors'' Voice','c16','28'),
 ('p5','Swords to Plowshares','sta','10');
INSERT INTO collections (id,owner_id,name,description,type,is_public) VALUES
 (10,1,'Box A',NULL,'mtg_cards',0),
 (11,1,'Box B','shelf 2','mtg_cards',0),
 (12,1,'Atraxa','in red deckbox','mtg_deck',1),
 (13,1,'Sealed',NULL,'mtg_sealed',0),
 (20,2,'B deck',NULL,'mtg_deck',0),
 (30,3,'Books',NULL,'books',0);
INSERT INTO card_items (collection_id,scryfall_card_id,foil_treatment,category,quantity,notes) VALUES
 (10,'p1','nonfoil','none',2,'nm'),
 (10,'p1','nonfoil','deck',1,'lp'),
 (10,'p1','foil','none',1,NULL),
 (11,'p3','nonfoil','none',4,NULL),
 (12,'p4','nonfoil','commander',1,NULL),
 (12,'p1','nonfoil','deck',1,NULL),
 (12,'p2','foil','deck',1,NULL),
 (12,'p3','nonfoil','none',1,NULL),
 (12,'p5','nonfoil','sideboard',2,NULL),
 (20,'p3','nonfoil','deck',1,NULL);
INSERT INTO sealed_items (collection_id,item_type,name,quantity) VALUES (13,'jumpstart_booster_box','JMP box',1);
`

func TestMigration009CollectionsDecks(t *testing.T) {
	db, dsn := testdb.NewEmpty(t)
	m, closeFn, err := migrations.New(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer closeFn()

	if err := m.Migrate(8); err != nil {
		t.Fatalf("migrate to 8: %v", err)
	}
	exec(t, db, seedV8)
	if err := m.Migrate(9); err != nil {
		t.Fatalf("migrate to 9: %v", err)
	}

	t.Run("quantities preserved per user", func(t *testing.T) {
		got := map[int64]int64{}
		rows := query(t, db, `
			SELECT owner_id, SUM(q) FROM (
			  SELECT c.owner_id, cc.quantity q FROM collection_cards cc JOIN collections c ON c.id = cc.collection_id
			  UNION ALL
			  SELECT d.owner_id, dc.quantity FROM deck_cards dc JOIN decks d ON d.id = dc.deck_id
			) t GROUP BY owner_id`)
		for rows.Next() {
			var owner, qty int64
			if err := rows.Scan(&owner, &qty); err != nil {
				t.Fatal(err)
			}
			got[owner] = qty
		}
		if got[1] != 14 || got[2] != 1 || len(got) != 2 {
			t.Errorf("quantities by owner = %v, want map[1:14 2:1]", got)
		}
	})

	t.Run("deck collections became decks", func(t *testing.T) {
		if n := count(t, db, `SELECT COUNT(*) FROM decks`); n != 2 {
			t.Errorf("decks = %d, want 2", n)
		}
		if n := count(t, db, `SELECT COUNT(*) FROM collections WHERE id IN (12, 20)`); n != 0 {
			t.Errorf("old deck collections remaining = %d, want 0", n)
		}
		var notes string
		var public bool
		if err := db.QueryRow(`SELECT notes, is_public FROM decks WHERE name = 'Atraxa'`).Scan(&notes, &public); err != nil {
			t.Fatal(err)
		}
		if notes != "in red deckbox" || !public {
			t.Errorf("Atraxa deck notes=%q public=%v", notes, public)
		}
	})

	t.Run("deck cards merged by name and board", func(t *testing.T) {
		want := map[string]int{
			"commander|Atraxa, Praetors' Voice": 1,
			"main|City of Brass":                2, // two printings, one nonfoil one foil
			"main|Sol Ring":                     1, // category none -> main
			"sideboard|Swords to Plowshares":    2,
		}
		rows := query(t, db, `SELECT dc.board, dc.card_name, dc.quantity FROM deck_cards dc JOIN decks d ON d.id = dc.deck_id WHERE d.name = 'Atraxa'`)
		got := map[string]int{}
		for rows.Next() {
			var board, name string
			var qty int
			if err := rows.Scan(&board, &name, &qty); err != nil {
				t.Fatal(err)
			}
			got[board+"|"+name] = qty
		}
		if len(got) != len(want) {
			t.Errorf("deck cards = %v, want %v", got, want)
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("%s = %d, want %d", k, got[k], v)
			}
		}
	})

	t.Run("owned cards merged across categories", func(t *testing.T) {
		var qty int
		var notes sql.NullString
		if err := db.QueryRow(`SELECT quantity, notes FROM collection_cards WHERE collection_id = 10 AND scryfall_card_id = 'p1' AND foil_treatment = 'nonfoil'`).Scan(&qty, &notes); err != nil {
			t.Fatal(err)
		}
		if qty != 3 || notes.String != "nm\nlp" {
			t.Errorf("merged row qty=%d notes=%q, want 3 %q", qty, notes.String, "nm\nlp")
		}
		if n := count(t, db, `SELECT COUNT(*) FROM collection_cards WHERE language <> 'en'`); n != 0 {
			t.Errorf("non-en rows = %d, want 0", n)
		}
	})

	t.Run("one default collection per user", func(t *testing.T) {
		// User 1's oldest card collection; users 2 and 3 had none, so get a new one.
		rows := query(t, db, `SELECT owner_id, id, name FROM collections WHERE is_default = 1 ORDER BY owner_id`)
		var got []string
		for rows.Next() {
			var owner, id int64
			var name string
			if err := rows.Scan(&owner, &id, &name); err != nil {
				t.Fatal(err)
			}
			got = append(got, name)
			if owner == 1 && id != 10 {
				t.Errorf("user 1 default = %d, want 10", id)
			}
		}
		if len(got) != 3 || got[1] != "Default" || got[2] != "Default" {
			t.Errorf("defaults = %v", got)
		}
		if _, err := db.Exec(`INSERT INTO collections (owner_id, name, type, is_default) VALUES (1, 'dup', 'mtg_cards', 1)`); err == nil {
			t.Error("second default collection was allowed")
		}
	})

	t.Run("sealed untouched", func(t *testing.T) {
		if n := count(t, db, `SELECT COUNT(*) FROM sealed_items WHERE collection_id = 13`); n != 1 {
			t.Errorf("sealed items = %d, want 1", n)
		}
	})

	t.Run("down and up again", func(t *testing.T) {
		if err := m.Steps(-1); err != nil {
			t.Fatalf("down: %v", err)
		}
		if n := count(t, db, `SELECT COALESCE(SUM(quantity), 0) FROM card_items`); n != 15 {
			t.Errorf("card_items quantity after down = %d, want 15", n)
		}
		if n := count(t, db, `SELECT COUNT(*) FROM collections WHERE type = 'mtg_deck'`); n != 2 {
			t.Errorf("deck collections after down = %d, want 2", n)
		}
		if err := m.Steps(1); err != nil {
			t.Fatalf("up again: %v", err)
		}
		if n := count(t, db, `SELECT COUNT(*) FROM decks`); n != 2 {
			t.Errorf("decks after re-up = %d, want 2", n)
		}
	})
}

func exec(t *testing.T, db *sql.DB, script string) {
	t.Helper()
	// The test connection doesn't enable multiStatements; run one at a time.
	for _, stmt := range splitStatements(script) {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("exec %q: %v", stmt, err)
		}
	}
}

// splitStatements splits on semicolons at line ends, which is enough for the
// seed above.
func splitStatements(script string) []string {
	var out []string
	var cur strings.Builder
	for _, line := range strings.Split(script, "\n") {
		cur.WriteString(line + "\n")
		if strings.HasSuffix(line, ";") {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	return out
}

func query(t *testing.T, db *sql.DB, q string) *sql.Rows {
	t.Helper()
	rows, err := db.Query(q)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	t.Cleanup(func() { rows.Close() })
	return rows
}

func count(t *testing.T, db *sql.DB, q string) int64 {
	t.Helper()
	var n int64
	if err := db.QueryRow(q).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}
