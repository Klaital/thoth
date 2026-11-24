//go:build integration

package handler

import (
	"context"
	"database/sql"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/klaital/thoth/internal/auth"
	"github.com/klaital/thoth/internal/db"
	thothv1 "github.com/klaital/thoth/internal/gen/thoth/v1"
	"github.com/klaital/thoth/internal/testdb"
)

const (
	alice = int64(1)
	bob   = int64(2)
)

// fixture sets up a migrated database with two users and a few Scryfall cards.
func fixture(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	sqlDB := testdb.New(t)
	for _, stmt := range []string{
		`INSERT INTO users (id, email) VALUES (1, 'alice@example.com'), (2, 'bob@example.com')`,
		// prices_json is never SQL NULL in synced data (the syncer writes
		// JSON), and json.RawMessage can't scan NULL.
		`INSERT INTO scryfall_cards (id, name, set_code, collector_number, image_uri, prices_json) VALUES
			('cob-mh2', 'City of Brass', 'mh2', '259', 'https://img/cob-mh2', '{}'),
			('cob-5ed', 'City of Brass', '5ed', '412', 'https://img/cob-5ed', '{}'),
			('sol-c21', 'Sol Ring', 'c21', '263', 'https://img/sol', '{}'),
			('atx-c16', 'Atraxa, Praetors'' Voice', 'c16', '28', 'https://img/atraxa', '{}'),
			('stp-sta', 'Swords to Plowshares', 'sta', '10', 'https://img/stp', '{}'),
			('dlv-isd', 'Delver of Secrets // Insectile Aberration', 'isd', '51', 'https://img/delver', '{}')`,
	} {
		if _, err := sqlDB.Exec(stmt); err != nil {
			t.Fatalf("seeding: %v", err)
		}
	}
	return sqlDB, db.New(sqlDB)
}

func as(user int64) context.Context {
	return auth.ContextWithUserID(context.Background(), user)
}

func wantCode(t *testing.T, err error, code codes.Code) {
	t.Helper()
	if status.Code(err) != code {
		t.Fatalf("got %v, want %s", err, code)
	}
}

func must[T any](t *testing.T) func(T, error) T {
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return v
	}
}

// fakeStream collects what a server-streaming handler sends.
type fakeStream[T any] struct {
	grpc.ServerStream
	ctx  context.Context
	sent []*T
}

func (f *fakeStream[T]) Context() context.Context { return f.ctx }
func (f *fakeStream[T]) Send(m *T) error         { f.sent = append(f.sent, m); return nil }

func TestCollectionsAndCardsIntegration(t *testing.T) {
	sqlDB, q := fixture(t)
	colls := NewCollectionHandler(sqlDB, q)
	cards := NewCardItemHandler(q)

	// Default collection is created on first listing, exactly once.
	list := must[*thothv1.ListCollectionsResponse](t)(colls.ListCollections(as(alice), &thothv1.ListCollectionsRequest{}))
	list = must[*thothv1.ListCollectionsResponse](t)(colls.ListCollections(as(alice), &thothv1.ListCollectionsRequest{}))
	if len(list.Collections) != 1 || !list.Collections[0].IsDefault || list.Collections[0].Name != "Default" {
		t.Fatalf("collections = %v, want one default", list.Collections)
	}
	def := list.Collections[0]

	_, err := colls.DeleteCollection(as(alice), &thothv1.DeleteCollectionRequest{Id: def.Id})
	wantCode(t, err, codes.FailedPrecondition)

	box := must[*thothv1.CreateCollectionResponse](t)(colls.CreateCollection(as(alice), &thothv1.CreateCollectionRequest{
		Name: "Box", Type: thothv1.CollectionType_COLLECTION_TYPE_MTG_CARDS,
	})).Collection
	if box.IsDefault {
		t.Fatal("new collection is default")
	}

	add := func(t *testing.T, coll int64, lang string, qty int32) *thothv1.CardItem {
		t.Helper()
		return must[*thothv1.AddCardItemResponse](t)(cards.AddCardItem(as(alice), &thothv1.AddCardItemRequest{
			CollectionId: coll, ScryfallCardId: "cob-mh2", Quantity: qty, Language: lang,
		})).Item
	}

	t.Run("language is part of the card's identity", func(t *testing.T) {
		en := add(t, box.Id, "", 2)
		ja := add(t, box.Id, "JA", 1)
		again := add(t, box.Id, "en", 1)
		if en.Language != "en" || ja.Language != "ja" {
			t.Errorf("languages = %q, %q", en.Language, ja.Language)
		}
		if again.Id != en.Id || again.Quantity != 3 {
			t.Errorf("re-add en: id %d qty %d, want id %d qty 3", again.Id, again.Quantity, en.Id)
		}
		_, err := cards.AddCardItem(as(alice), &thothv1.AddCardItemRequest{CollectionId: box.Id, ScryfallCardId: "cob-mh2", Language: "english"})
		wantCode(t, err, codes.InvalidArgument)

		// Changing ja -> en merges into the en row.
		merged := must[*thothv1.UpdateCardItemResponse](t)(cards.UpdateCardItem(as(alice), &thothv1.UpdateCardItemRequest{
			Id: ja.Id, Quantity: ja.Quantity, Language: proto.String("en"),
		})).Item
		if merged.Id != en.Id || merged.Quantity != 4 {
			t.Errorf("merged = id %d qty %d, want id %d qty 4", merged.Id, merged.Quantity, en.Id)
		}
	})

	t.Run("listing cards enforces read access", func(t *testing.T) {
		_, err := cards.ListCardItems(as(bob), &thothv1.ListCardItemsRequest{CollectionId: box.Id})
		wantCode(t, err, codes.NotFound)

		must[*thothv1.UpdateCollectionResponse](t)(colls.UpdateCollection(as(alice), &thothv1.UpdateCollectionRequest{Id: box.Id, Name: "Box", IsPublic: true}))
		got := must[*thothv1.ListCardItemsResponse](t)(cards.ListCardItems(as(bob), &thothv1.ListCardItemsRequest{CollectionId: box.Id}))
		if len(got.Items) != 1 {
			t.Errorf("public listing = %d items, want 1", len(got.Items))
		}
	})

	t.Run("move merges on printing, foil and language", func(t *testing.T) {
		add(t, def.Id, "en", 1)
		src := must[*thothv1.ListCardItemsResponse](t)(cards.ListCardItems(as(alice), &thothv1.ListCardItemsRequest{CollectionId: box.Id}))
		must[*thothv1.MoveCollectionItemsResponse](t)(colls.MoveCollectionItems(as(alice), &thothv1.MoveCollectionItemsRequest{
			CollectionId: box.Id, ItemIds: []int64{src.Items[0].Id}, TargetCollectionId: def.Id,
		}))
		dst := must[*thothv1.ListCardItemsResponse](t)(cards.ListCardItems(as(alice), &thothv1.ListCardItemsRequest{CollectionId: def.Id}))
		if len(dst.Items) != 1 || dst.Items[0].Quantity != 5 {
			t.Errorf("default after move = %v, want one row of 5", dst.Items)
		}

		sealed := must[*thothv1.CreateCollectionResponse](t)(colls.CreateCollection(as(alice), &thothv1.CreateCollectionRequest{
			Name: "Sealed", Type: thothv1.CollectionType_COLLECTION_TYPE_MTG_SEALED,
		})).Collection
		_, err := colls.MoveCollectionItems(as(alice), &thothv1.MoveCollectionItemsRequest{
			CollectionId: def.Id, ItemIds: []int64{dst.Items[0].Id}, TargetCollectionId: sealed.Id,
		})
		wantCode(t, err, codes.InvalidArgument)
	})

	t.Run("move keeps different language and foil separate", func(t *testing.T) {
		// Default now holds 5 English non-foil City of Brass (from the test above).
		other := must[*thothv1.CreateCollectionResponse](t)(colls.CreateCollection(as(alice), &thothv1.CreateCollectionRequest{
			Name: "Binder", Type: thothv1.CollectionType_COLLECTION_TYPE_MTG_CARDS,
		})).Collection
		ja := add(t, other.Id, "ja", 1)
		foil := must[*thothv1.AddCardItemResponse](t)(cards.AddCardItem(as(alice), &thothv1.AddCardItemRequest{
			CollectionId: other.Id, ScryfallCardId: "cob-mh2", FoilTreatment: thothv1.FoilTreatment_FOIL_TREATMENT_FOIL, Quantity: 2,
		})).Item
		must[*thothv1.MoveCollectionItemsResponse](t)(colls.MoveCollectionItems(as(alice), &thothv1.MoveCollectionItemsRequest{
			CollectionId: other.Id, ItemIds: []int64{ja.Id, foil.Id}, TargetCollectionId: def.Id,
		}))

		dst := must[*thothv1.ListCardItemsResponse](t)(cards.ListCardItems(as(alice), &thothv1.ListCardItemsRequest{CollectionId: def.Id}))
		got := map[string]int32{}
		for _, it := range dst.Items {
			got[it.Language+"/"+it.FoilTreatment.String()] = it.Quantity
		}
		want := map[string]int32{
			"en/FOIL_TREATMENT_NONFOIL": 5,
			"ja/FOIL_TREATMENT_NONFOIL": 1,
			"en/FOIL_TREATMENT_FOIL":    2,
		}
		if len(got) != len(want) {
			t.Fatalf("default after move = %v, want %v", got, want)
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("%s = %d, want %d", k, got[k], v)
			}
		}
		left := must[*thothv1.ListCardItemsResponse](t)(cards.ListCardItems(as(alice), &thothv1.ListCardItemsRequest{CollectionId: other.Id}))
		if len(left.Items) != 0 {
			t.Errorf("source still has %d items", len(left.Items))
		}
	})

	t.Run("bulk import records language and skips deck headers", func(t *testing.T) {
		imp := must[*thothv1.CreateCollectionResponse](t)(colls.CreateCollection(as(alice), &thothv1.CreateCollectionRequest{
			Name: "German box", Type: thothv1.CollectionType_COLLECTION_TYPE_MTG_CARDS,
		})).Collection
		text := "Commander\n" +
			"2 City of Brass (MH2) 259\n" +
			"Sideboard:\n" +
			"1 Sol Ring (C21) 263 *F*\n"
		stream := &fakeStream[thothv1.BulkImportCardLineResult]{ctx: as(alice)}
		if err := cards.BulkImportCardItems(&thothv1.BulkImportCardItemsRequest{CollectionId: imp.Id, Text: text, Language: "DE"}, stream); err != nil {
			t.Fatal(err)
		}
		// Header lines produce no results; both card lines are added.
		if len(stream.sent) != 2 {
			t.Fatalf("got %d results, want 2: %v", len(stream.sent), stream.sent)
		}
		for i, wantLine := range []int32{2, 4} {
			r := stream.sent[i]
			if r.LineNumber != wantLine || r.Error != "" || r.Item == nil {
				t.Fatalf("result %d = line %d error %q item %v, want line %d added", i, r.LineNumber, r.Error, r.Item, wantLine)
			}
			if r.Item.Language != "de" {
				t.Errorf("line %d language = %q, want de", wantLine, r.Item.Language)
			}
		}
		if f := stream.sent[1].Item.FoilTreatment; f != thothv1.FoilTreatment_FOIL_TREATMENT_FOIL {
			t.Errorf("Sol Ring foil = %s, want FOIL", f)
		}

		listed := must[*thothv1.ListCardItemsResponse](t)(cards.ListCardItems(as(alice), &thothv1.ListCardItemsRequest{CollectionId: imp.Id}))
		if len(listed.Items) != 2 {
			t.Errorf("collection has %d rows, want 2", len(listed.Items))
		}

		err := cards.BulkImportCardItems(&thothv1.BulkImportCardItemsRequest{CollectionId: imp.Id, Text: text, Language: "german"},
			&fakeStream[thothv1.BulkImportCardLineResult]{ctx: as(alice)})
		wantCode(t, err, codes.InvalidArgument)
	})
}

func TestDecksIntegration(t *testing.T) {
	_, q := fixture(t)
	decks := NewDeckHandler(q)

	_, err := decks.CreateDeck(as(alice), &thothv1.CreateDeckRequest{Name: "  "})
	wantCode(t, err, codes.InvalidArgument)

	deck := must[*thothv1.CreateDeckResponse](t)(decks.CreateDeck(as(alice), &thothv1.CreateDeckRequest{
		Name: "Atraxa", Notes: "red deckbox",
	})).Deck
	if deck.Name != "Atraxa" || deck.Notes != "red deckbox" || deck.IsPublic {
		t.Fatalf("created deck = %+v", deck)
	}

	addByName := func(name string, board thothv1.DeckBoard, qty int32) (*thothv1.DeckCard, error) {
		res, err := decks.AddDeckCard(as(alice), &thothv1.AddDeckCardRequest{DeckId: deck.Id, CardName: name, Board: board, Quantity: qty})
		if err != nil {
			return nil, err
		}
		return res.Card, nil
	}

	t.Run("cards are added by name", func(t *testing.T) {
		cob := must[*thothv1.DeckCard](t)(addByName("city of brass", thothv1.DeckBoard_DECK_BOARD_UNSPECIFIED, 0))
		if cob.CardName != "City of Brass" || cob.Board != thothv1.DeckBoard_DECK_BOARD_MAIN || cob.Quantity != 1 {
			t.Errorf("added %+v", cob)
		}
		if cob.ImageUri == "" || cob.PreferredScryfallCardId != "" {
			t.Errorf("image %q (want any printing's), printing %q (want none)", cob.ImageUri, cob.PreferredScryfallCardId)
		}
		again := must[*thothv1.DeckCard](t)(addByName("City of Brass", thothv1.DeckBoard_DECK_BOARD_MAIN, 2))
		if again.Id != cob.Id || again.Quantity != 3 {
			t.Errorf("re-add = id %d qty %d, want id %d qty 3", again.Id, again.Quantity, cob.Id)
		}

		dfc := must[*thothv1.DeckCard](t)(addByName("Delver of Secrets", thothv1.DeckBoard_DECK_BOARD_MAIN, 1))
		if dfc.CardName != "Delver of Secrets // Insectile Aberration" {
			t.Errorf("front face resolved to %q", dfc.CardName)
		}

		_, err := addByName("Not A Real Card", thothv1.DeckBoard_DECK_BOARD_MAIN, 1)
		wantCode(t, err, codes.InvalidArgument)
		_, err = addByName("City of %", thothv1.DeckBoard_DECK_BOARD_MAIN, 1)
		wantCode(t, err, codes.InvalidArgument)
	})

	t.Run("preferred printing and commanders", func(t *testing.T) {
		cmd := must[*thothv1.AddDeckCardResponse](t)(decks.AddDeckCard(as(alice), &thothv1.AddDeckCardRequest{
			DeckId: deck.Id, PreferredScryfallCardId: "atx-c16", Board: thothv1.DeckBoard_DECK_BOARD_COMMANDER,
		})).Card
		if cmd.CardName != "Atraxa, Praetors' Voice" || cmd.PreferredSetCode != "c16" || cmd.ImageUri != "https://img/atraxa" {
			t.Errorf("commander = %+v", cmd)
		}
		got := must[*thothv1.GetDeckResponse](t)(decks.GetDeck(as(alice), &thothv1.GetDeckRequest{Id: deck.Id})).Deck
		if len(got.Commanders) != 1 || got.Commanders[0].CardName != "Atraxa, Praetors' Voice" {
			t.Errorf("commanders = %v", got.Commanders)
		}
	})

	t.Run("update board merges and printing must match name", func(t *testing.T) {
		side := must[*thothv1.DeckCard](t)(addByName("Sol Ring", thothv1.DeckBoard_DECK_BOARD_SIDEBOARD, 1))
		main := must[*thothv1.DeckCard](t)(addByName("Sol Ring", thothv1.DeckBoard_DECK_BOARD_MAIN, 2))
		merged := must[*thothv1.UpdateDeckCardResponse](t)(decks.UpdateDeckCard(as(alice), &thothv1.UpdateDeckCardRequest{
			Id: main.Id, Board: thothv1.DeckBoard_DECK_BOARD_SIDEBOARD.Enum(),
		})).Card
		if merged.Id != side.Id || merged.Quantity != 3 {
			t.Errorf("merged = id %d qty %d, want id %d qty 3", merged.Id, merged.Quantity, side.Id)
		}

		_, err := decks.UpdateDeckCard(as(alice), &thothv1.UpdateDeckCardRequest{Id: side.Id, PreferredScryfallCardId: proto.String("cob-mh2")})
		wantCode(t, err, codes.InvalidArgument)
		withPrinting := must[*thothv1.UpdateDeckCardResponse](t)(decks.UpdateDeckCard(as(alice), &thothv1.UpdateDeckCardRequest{
			Id: side.Id, PreferredScryfallCardId: proto.String("sol-c21"),
		})).Card
		if withPrinting.PreferredCollectorNumber != "263" || withPrinting.Quantity != 3 {
			t.Errorf("after setting printing: %+v", withPrinting)
		}
		_, err = decks.UpdateDeckCard(as(alice), &thothv1.UpdateDeckCardRequest{Id: side.Id, Quantity: proto.Int32(0)})
		wantCode(t, err, codes.InvalidArgument)
	})

	t.Run("other users", func(t *testing.T) {
		_, err := decks.GetDeck(as(bob), &thothv1.GetDeckRequest{Id: deck.Id})
		wantCode(t, err, codes.NotFound)
		_, err = decks.ListDeckCards(as(bob), &thothv1.ListDeckCardsRequest{DeckId: deck.Id})
		wantCode(t, err, codes.NotFound)
		_, err = decks.UpdateDeck(as(bob), &thothv1.UpdateDeckRequest{Id: deck.Id, Name: "mine now"})
		wantCode(t, err, codes.PermissionDenied)
		_, err = decks.AddDeckCard(as(bob), &thothv1.AddDeckCardRequest{DeckId: deck.Id, CardName: "Sol Ring"})
		wantCode(t, err, codes.PermissionDenied)

		bobs := must[*thothv1.ListDecksResponse](t)(decks.ListDecks(as(bob), &thothv1.ListDecksRequest{OwnerId: proto.Int64(alice)}))
		if len(bobs.Decks) != 0 {
			t.Errorf("bob sees %d of alice's private decks", len(bobs.Decks))
		}
	})

	t.Run("owner edits name, notes and visibility", func(t *testing.T) {
		updated := must[*thothv1.UpdateDeckResponse](t)(decks.UpdateDeck(as(alice), &thothv1.UpdateDeckRequest{
			Id: deck.Id, Name: "  Atraxa Superfriends ", Notes: "Moved to the blue box.\nNeeds more proliferate.", IsPublic: true,
		})).Deck
		if updated.Name != "Atraxa Superfriends" || updated.Notes != "Moved to the blue box.\nNeeds more proliferate." || !updated.IsPublic {
			t.Errorf("updated deck = %+v", updated)
		}
		// Commanders are still attached after an update.
		if len(updated.Commanders) != 1 {
			t.Errorf("commanders after update = %v", updated.Commanders)
		}

		// Stored, and now visible to other users because it's public.
		seen := must[*thothv1.GetDeckResponse](t)(decks.GetDeck(as(bob), &thothv1.GetDeckRequest{Id: deck.Id})).Deck
		if seen.Name != updated.Name || seen.Notes != updated.Notes {
			t.Errorf("bob sees %+v", seen)
		}
		if _, err := decks.ListDeckCards(as(bob), &thothv1.ListDeckCardsRequest{DeckId: deck.Id}); err != nil {
			t.Errorf("bob listing public deck cards: %v", err)
		}

		_, err := decks.UpdateDeck(as(alice), &thothv1.UpdateDeckRequest{Id: deck.Id, Name: " "})
		wantCode(t, err, codes.InvalidArgument)

		// Clearing notes is allowed.
		cleared := must[*thothv1.UpdateDeckResponse](t)(decks.UpdateDeck(as(alice), &thothv1.UpdateDeckRequest{
			Id: deck.Id, Name: "Atraxa Superfriends", IsPublic: true,
		})).Deck
		if cleared.Notes != "" {
			t.Errorf("notes after clearing = %q", cleared.Notes)
		}
	})

	t.Run("bulk import", func(t *testing.T) {
		fresh := must[*thothv1.CreateDeckResponse](t)(decks.CreateDeck(as(alice), &thothv1.CreateDeckRequest{Name: "Imported"})).Deck
		text := "Commander\n" +
			"1 Atraxa, Praetors' Voice (C16) 28\n" +
			"Deck\n" +
			"2 Sol Ring\n" +
			"\n" +
			"1 Bogus Card\n" +
			"1 City of Brass (XXX) 1\n" +
			"Sideboard:\n" +
			"Swords to Plowshares\n"
		stream := &fakeStream[thothv1.BulkImportDeckCardLineResult]{ctx: as(alice)}
		if err := decks.BulkImportDeckCards(&thothv1.BulkImportDeckCardsRequest{DeckId: fresh.Id, Text: text}, stream); err != nil {
			t.Fatal(err)
		}
		type result struct {
			line          int32
			name          string
			board         thothv1.DeckBoard
			qty           int32
			warning, fail bool
		}
		want := []result{
			{2, "Atraxa, Praetors' Voice", thothv1.DeckBoard_DECK_BOARD_COMMANDER, 1, false, false},
			{4, "Sol Ring", thothv1.DeckBoard_DECK_BOARD_MAIN, 2, false, false},
			{6, "", 0, 0, false, true},
			{7, "City of Brass", thothv1.DeckBoard_DECK_BOARD_MAIN, 1, true, false},
			{9, "Swords to Plowshares", thothv1.DeckBoard_DECK_BOARD_SIDEBOARD, 1, false, false},
		}
		if len(stream.sent) != len(want) {
			t.Fatalf("got %d results, want %d: %v", len(stream.sent), len(want), stream.sent)
		}
		for i, w := range want {
			r := stream.sent[i]
			got := result{line: r.LineNumber, warning: r.Warning != "", fail: r.Error != ""}
			if r.Card != nil {
				got.name, got.board, got.qty = r.Card.CardName, r.Card.Board, r.Card.Quantity
			}
			if got != w {
				t.Errorf("result %d = %+v (warning %q, error %q), want %+v", i, got, r.Warning, r.Error, w)
			}
		}
		if stream.sent[0].Card.PreferredScryfallCardId != "atx-c16" {
			t.Errorf("Moxfield line printing = %q, want atx-c16", stream.sent[0].Card.PreferredScryfallCardId)
		}
	})

	t.Run("delete cascades", func(t *testing.T) {
		must[*thothv1.DeleteDeckResponse](t)(decks.DeleteDeck(as(alice), &thothv1.DeleteDeckRequest{Id: deck.Id}))
		_, err := decks.GetDeck(as(alice), &thothv1.GetDeckRequest{Id: deck.Id})
		wantCode(t, err, codes.NotFound)
		cards, err := q.ListDeckCards(context.Background(), uint64(deck.Id))
		if err != nil || len(cards) != 0 {
			t.Errorf("deck cards after delete = %d (err %v)", len(cards), err)
		}
	})
}
