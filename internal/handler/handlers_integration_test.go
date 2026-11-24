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
	cards := NewCardItemHandler(sqlDB, q)

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

	t.Run("re-adding a card keeps its notes unless new ones are given", func(t *testing.T) {
		binder := must[*thothv1.CreateCollectionResponse](t)(colls.CreateCollection(as(alice), &thothv1.CreateCollectionRequest{
			Name: "Notes binder", Type: thothv1.CollectionType_COLLECTION_TYPE_MTG_CARDS,
		})).Collection
		addSol := func(t *testing.T, notes string) *thothv1.CardItem {
			t.Helper()
			return must[*thothv1.AddCardItemResponse](t)(cards.AddCardItem(as(alice), &thothv1.AddCardItemRequest{
				CollectionId: binder.Id, ScryfallCardId: "sol-c21", Quantity: 1, Notes: notes,
			})).Item
		}

		if got := addSol(t, "page 3").Notes; got != "page 3" {
			t.Fatalf("new card notes = %q, want %q", got, "page 3")
		}
		if got := addSol(t, "").Notes; got != "page 3" {
			t.Errorf("notes after re-add without notes = %q, want kept", got)
		}

		stream := &fakeStream[thothv1.BulkImportCardLineResult]{ctx: as(alice)}
		if err := cards.BulkImportCardItems(&thothv1.BulkImportCardItemsRequest{CollectionId: binder.Id, Text: "1 Sol Ring (C21) 263\n"}, stream); err != nil {
			t.Fatal(err)
		}
		if r := stream.sent[0]; r.Item == nil || r.Item.Notes != "page 3" || r.Item.Quantity != 3 {
			t.Errorf("after bulk re-import: item %v, want notes kept and qty 3", r.Item)
		}

		if got := addSol(t, "page 4").Notes; got != "page 4" {
			t.Errorf("notes after re-add with new notes = %q, want %q", got, "page 4")
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

	t.Run("bulk import batches printings and merges repeats", func(t *testing.T) {
		imp := must[*thothv1.CreateCollectionResponse](t)(colls.CreateCollection(as(alice), &thothv1.CreateCollectionRequest{
			Name: "Batched box", Type: thothv1.CollectionType_COLLECTION_TYPE_MTG_CARDS,
		})).Collection
		// A German Sol Ring already in the collection must stay a separate row.
		de := "1 Sol Ring (C21) 263\n"
		if err := cards.BulkImportCardItems(&thothv1.BulkImportCardItemsRequest{CollectionId: imp.Id, Text: de, Language: "de"},
			&fakeStream[thothv1.BulkImportCardLineResult]{ctx: as(alice)}); err != nil {
			t.Fatal(err)
		}

		text := "1 Sol Ring (c21) 263\n" +
			"1 Sol Ring (C21) 263 *F*\n" +
			"2 sol ring (C21) 263\n" +
			"1 Black Lotus (LEA) 232\n" +
			"1 City of Brass (C21) 263\n" +
			"not a card line\n"
		stream := &fakeStream[thothv1.BulkImportCardLineResult]{ctx: as(alice)}
		if err := cards.BulkImportCardItems(&thothv1.BulkImportCardItemsRequest{CollectionId: imp.Id, Text: text}, stream); err != nil {
			t.Fatal(err)
		}
		// Each added line echoes its stored row, so merged lines show the
		// total: three nonfoil lines (one with a name warning) make 4.
		type result struct {
			qty           int32
			foil          bool
			warning, fail bool
		}
		want := []result{
			{4, false, false, false},
			{1, true, false, false},
			{4, false, false, false},
			{0, false, false, true},
			{4, false, true, false},
			{0, false, false, true},
		}
		if len(stream.sent) != len(want) {
			t.Fatalf("got %d results, want %d: %v", len(stream.sent), len(want), stream.sent)
		}
		for i, w := range want {
			r := stream.sent[i]
			got := result{warning: r.Warning != "", fail: r.Error != ""}
			if r.Item != nil {
				if r.Item.Language != "en" {
					t.Errorf("result %d language = %q, want en", i, r.Item.Language)
				}
				got.qty = r.Item.Quantity
				got.foil = r.Item.FoilTreatment == thothv1.FoilTreatment_FOIL_TREATMENT_FOIL
			}
			if got != w {
				t.Errorf("result %d = %+v (warning %q, error %q), want %+v", i, got, r.Warning, r.Error, w)
			}
		}

		listed := must[*thothv1.ListCardItemsResponse](t)(cards.ListCardItems(as(alice), &thothv1.ListCardItemsRequest{CollectionId: imp.Id}))
		if len(listed.Items) != 3 {
			t.Errorf("collection has %d rows, want 3 (de, en nonfoil, en foil): %v", len(listed.Items), listed.Items)
		}
	})
}

func TestDecksIntegration(t *testing.T) {
	sqlDB, q := fixture(t)
	decks := NewDeckHandler(sqlDB, q)

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

	t.Run("bulk import batches names and merges repeats", func(t *testing.T) {
		fresh := must[*thothv1.CreateDeckResponse](t)(decks.CreateDeck(as(alice), &thothv1.CreateDeckRequest{Name: "Batched"})).Deck
		// Case-folded names, a double-faced front name (the one-at-a-time
		// fallback), and repeats that merge into one row per name and board.
		text := "1 sol ring\n" +
			"1 Delver of Secrets\n" +
			"2 Sol Ring\n" +
			"1 Swords to Plowshares (STA) 10\n" +
			"1 swords to plowshares\n"
		stream := &fakeStream[thothv1.BulkImportDeckCardLineResult]{ctx: as(alice)}
		if err := decks.BulkImportDeckCards(&thothv1.BulkImportDeckCardsRequest{DeckId: fresh.Id, Text: text}, stream); err != nil {
			t.Fatal(err)
		}
		// Each line echoes the stored row, so merged lines show the total.
		want := []struct {
			name string
			qty  int32
		}{
			{"Sol Ring", 3},
			{"Delver of Secrets // Insectile Aberration", 1},
			{"Sol Ring", 3},
			{"Swords to Plowshares", 2},
			{"Swords to Plowshares", 2},
		}
		if len(stream.sent) != len(want) {
			t.Fatalf("got %d results, want %d: %v", len(stream.sent), len(want), stream.sent)
		}
		for i, w := range want {
			r := stream.sent[i]
			if r.Error != "" || r.Warning != "" || r.Card == nil {
				t.Errorf("result %d: error %q, warning %q, card %v", i, r.Error, r.Warning, r.Card)
				continue
			}
			if r.Card.CardName != w.name || r.Card.Quantity != w.qty {
				t.Errorf("result %d = %q x%d, want %q x%d", i, r.Card.CardName, r.Card.Quantity, w.name, w.qty)
			}
		}
		if got := stream.sent[3].Card.PreferredScryfallCardId; got != "stp-sta" {
			t.Errorf("Moxfield line printing = %q, want stp-sta", got)
		}
		cards := must[*thothv1.ListDeckCardsResponse](t)(decks.ListDeckCards(as(alice), &thothv1.ListDeckCardsRequest{DeckId: fresh.Id})).Cards
		if len(cards) != 3 {
			t.Errorf("deck has %d cards, want 3: %v", len(cards), cards)
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

func TestDeckCheckoutIntegration(t *testing.T) {
	sqlDB, q := fixture(t)
	colls := NewCollectionHandler(sqlDB, q)
	cards := NewCardItemHandler(sqlDB, q)
	decks := NewDeckHandler(sqlDB, q)

	newBox := func(owner int64, name string) int64 {
		return must[*thothv1.CreateCollectionResponse](t)(colls.CreateCollection(as(owner), &thothv1.CreateCollectionRequest{
			Name: name, Type: thothv1.CollectionType_COLLECTION_TYPE_MTG_CARDS,
		})).Collection.Id
	}
	addCard := func(coll int64, printing string, foil thothv1.FoilTreatment, lang string, qty int32) int64 {
		return must[*thothv1.AddCardItemResponse](t)(cards.AddCardItem(as(alice), &thothv1.AddCardItemRequest{
			CollectionId: coll, ScryfallCardId: printing, FoilTreatment: foil, Language: lang, Quantity: qty,
		})).Item.Id
	}
	newDeck := func(owner int64, name string, cardsByBoard map[thothv1.DeckBoard]map[string]int32) int64 {
		id := must[*thothv1.CreateDeckResponse](t)(decks.CreateDeck(as(owner), &thothv1.CreateDeckRequest{Name: name})).Deck.Id
		for board, byName := range cardsByBoard {
			for n, qty := range byName {
				must[*thothv1.AddDeckCardResponse](t)(decks.AddDeckCard(as(owner), &thothv1.AddDeckCardRequest{
					DeckId: id, CardName: n, Board: board, Quantity: qty,
				}))
			}
		}
		return id
	}
	ownership := func(deck int64) map[string]*thothv1.DeckCardOwnership {
		res := must[*thothv1.GetDeckOwnershipResponse](t)(decks.GetDeckOwnership(as(alice), &thothv1.GetDeckOwnershipRequest{DeckId: deck}))
		out := map[string]*thothv1.DeckCardOwnership{}
		for _, c := range res.Cards {
			out[c.CardName] = c
		}
		out[""] = &thothv1.DeckCardOwnership{Missing: res.TotalMissing, CheckedOutHere: res.TotalCheckedOut}
		return out
	}
	checkouts := func(coll int64) map[int64][]*thothv1.CardCheckout {
		res := must[*thothv1.ListCardItemsResponse](t)(cards.ListCardItems(as(alice), &thothv1.ListCardItemsRequest{CollectionId: coll}))
		out := map[int64][]*thothv1.CardCheckout{}
		for _, it := range res.Items {
			out[it.Id] = it.Checkouts
		}
		return out
	}
	const (
		nonfoil = thothv1.FoilTreatment_FOIL_TREATMENT_NONFOIL
		foil    = thothv1.FoilTreatment_FOIL_TREATMENT_FOIL
		main    = thothv1.DeckBoard_DECK_BOARD_MAIN
		side    = thothv1.DeckBoard_DECK_BOARD_SIDEBOARD
	)

	boxA, boxB := newBox(alice, "Box A"), newBox(alice, "Box B")
	cobEN := addCard(boxA, "cob-mh2", nonfoil, "en", 2)
	cobFoil := addCard(boxA, "cob-5ed", foil, "en", 1)
	cobJA := addCard(boxB, "cob-mh2", nonfoil, "ja", 1)
	sol := addCard(boxB, "sol-c21", nonfoil, "en", 1)

	atraxa := newDeck(alice, "Atraxa", map[thothv1.DeckBoard]map[string]int32{
		main: {"City of Brass": 3, "Sol Ring": 1, "Swords to Plowshares": 1},
		side: {"City of Brass": 1},
	})
	other := newDeck(alice, "Other", map[thothv1.DeckBoard]map[string]int32{main: {"Sol Ring": 1}})
	must[*thothv1.CheckOutDeckCardsResponse](t)(decks.CheckOutDeckCards(as(alice), &thothv1.CheckOutDeckCardsRequest{
		DeckId: other, Picks: []*thothv1.CheckoutPick{{CollectionCardId: sol, Quantity: 1}},
	}))

	t.Run("ownership counts any printing, foil and language across boxes", func(t *testing.T) {
		own := ownership(atraxa)
		cob := own["City of Brass"]
		if cob.Needed != 4 || cob.Free != 4 || cob.Missing != 0 || cob.CheckedOutHere != 0 || len(cob.Copies) != 3 {
			t.Errorf("City of Brass = needed %d free %d missing %d here %d copies %d, want 4/4/0/0/3",
				cob.Needed, cob.Free, cob.Missing, cob.CheckedOutHere, len(cob.Copies))
		}
		// Nonfoil English first, then nonfoil Japanese, then the foil.
		suggested := map[int64]int32{}
		for _, c := range cob.Copies {
			suggested[c.CollectionCardId] = c.SuggestedQuantity
		}
		if suggested[cobEN] != 2 || suggested[cobJA] != 1 || suggested[cobFoil] != 1 {
			t.Errorf("suggestions = %v", suggested)
		}
		if s := own["Sol Ring"]; s.Free != 0 || s.CheckedOutElsewhere != 1 || s.Missing != 1 {
			t.Errorf("Sol Ring = free %d elsewhere %d missing %d, want 0/1/1", s.Free, s.CheckedOutElsewhere, s.Missing)
		}
		if s := own["Swords to Plowshares"]; s.Missing != 1 || len(s.Copies) != 0 {
			t.Errorf("Swords = missing %d copies %d, want 1/0", s.Missing, len(s.Copies))
		}
		if own[""].Missing != 2 {
			t.Errorf("total missing = %d, want 2", own[""].Missing)
		}
	})

	t.Run("only the owner sees ownership and can check out", func(t *testing.T) {
		_, err := decks.GetDeckOwnership(as(bob), &thothv1.GetDeckOwnershipRequest{DeckId: atraxa})
		wantCode(t, err, codes.PermissionDenied)
		bobDeck := newDeck(bob, "Bob's", nil)
		_, err = decks.CheckOutDeckCards(as(bob), &thothv1.CheckOutDeckCardsRequest{
			DeckId: bobDeck, Picks: []*thothv1.CheckoutPick{{CollectionCardId: cobEN, Quantity: 1}},
		})
		wantCode(t, err, codes.NotFound)
	})

	t.Run("checking out more than is free records nothing", func(t *testing.T) {
		_, err := decks.CheckOutDeckCards(as(alice), &thothv1.CheckOutDeckCardsRequest{
			DeckId: atraxa, Picks: []*thothv1.CheckoutPick{{CollectionCardId: cobJA, Quantity: 1}, {CollectionCardId: sol, Quantity: 1}},
		})
		wantCode(t, err, codes.FailedPrecondition)
		if here := ownership(atraxa)[""].CheckedOutHere; here != 0 {
			t.Errorf("checked out after failed request = %d, want 0", here)
		}
	})

	// Partial check-out: the foil stays in the box.
	must[*thothv1.CheckOutDeckCardsResponse](t)(decks.CheckOutDeckCards(as(alice), &thothv1.CheckOutDeckCardsRequest{
		DeckId: atraxa, Picks: []*thothv1.CheckoutPick{{CollectionCardId: cobEN, Quantity: 2}, {CollectionCardId: cobJA, Quantity: 1}},
	}))

	t.Run("checked-out cards show the deck in collection views and search", func(t *testing.T) {
		cob := ownership(atraxa)["City of Brass"]
		if cob.CheckedOutHere != 3 || cob.Free != 1 || cob.Missing != 0 {
			t.Errorf("after check-out = here %d free %d missing %d, want 3/1/0", cob.CheckedOutHere, cob.Free, cob.Missing)
		}
		co := checkouts(boxA)
		if len(co[cobEN]) != 1 || co[cobEN][0].DeckName != "Atraxa" || co[cobEN][0].Quantity != 2 || len(co[cobFoil]) != 0 {
			t.Errorf("box A checkouts = %v", co)
		}
		found := must[*thothv1.SearchCollectionCardsResponse](t)(colls.SearchCollectionCards(as(alice), &thothv1.SearchCollectionCardsRequest{Query: "sol ring"}))
		if len(found.Matches) != 1 || len(found.Matches[0].Items[0].Checkouts) != 1 || found.Matches[0].Items[0].Checkouts[0].DeckName != "Other" {
			t.Errorf("search = %v", found.Matches)
		}
	})

	t.Run("quantity can't drop below the checked-out copies", func(t *testing.T) {
		_, err := cards.UpdateCardItem(as(alice), &thothv1.UpdateCardItemRequest{Id: cobEN, Quantity: 1})
		wantCode(t, err, codes.FailedPrecondition)
		must[*thothv1.UpdateCardItemResponse](t)(cards.UpdateCardItem(as(alice), &thothv1.UpdateCardItemRequest{Id: cobEN, Quantity: 3}))
	})

	t.Run("deck with checked-out cards can't be deleted", func(t *testing.T) {
		_, err := decks.DeleteDeck(as(alice), &thothv1.DeleteDeckRequest{Id: atraxa})
		wantCode(t, err, codes.FailedPrecondition)
	})

	t.Run("merging rows on move keeps their checkouts", func(t *testing.T) {
		boxC := newBox(alice, "Box C")
		extra := addCard(boxC, "cob-mh2", nonfoil, "en", 1)
		must[*thothv1.CheckOutDeckCardsResponse](t)(decks.CheckOutDeckCards(as(alice), &thothv1.CheckOutDeckCardsRequest{
			DeckId: atraxa, Picks: []*thothv1.CheckoutPick{{CollectionCardId: extra, Quantity: 1}},
		}))
		must[*thothv1.MoveCollectionItemsResponse](t)(colls.MoveCollectionItems(as(alice), &thothv1.MoveCollectionItemsRequest{
			CollectionId: boxC, ItemIds: []int64{extra}, TargetCollectionId: boxA,
		}))
		co := checkouts(boxA)[cobEN]
		if len(co) != 1 || co[0].Quantity != 3 {
			t.Errorf("merged row checkouts = %v, want 3 in Atraxa", co)
		}
	})

	t.Run("return goes box by box, partially then fully", func(t *testing.T) {
		list := must[*thothv1.ListDeckCheckoutsResponse](t)(decks.ListDeckCheckouts(as(alice), &thothv1.ListDeckCheckoutsRequest{DeckId: atraxa}))
		if len(list.Checkouts) != 2 || list.Checkouts[0].CollectionName != "Box A" || list.Checkouts[1].CollectionName != "Box B" {
			t.Fatalf("checkouts = %v", list.Checkouts)
		}
		must[*thothv1.ReturnDeckCardsResponse](t)(decks.ReturnDeckCards(as(alice), &thothv1.ReturnDeckCardsRequest{
			DeckId: atraxa, Picks: []*thothv1.ReturnPick{{CollectionCardId: cobEN, Quantity: 1}},
		}))
		if co := checkouts(boxA)[cobEN]; len(co) != 1 || co[0].Quantity != 2 {
			t.Errorf("after partial return = %v, want 2", co)
		}
		_, err := decks.ReturnDeckCards(as(alice), &thothv1.ReturnDeckCardsRequest{
			DeckId: atraxa, Picks: []*thothv1.ReturnPick{{CollectionCardId: cobFoil}},
		})
		wantCode(t, err, codes.NotFound)

		must[*thothv1.ReturnDeckCardsResponse](t)(decks.ReturnDeckCards(as(alice), &thothv1.ReturnDeckCardsRequest{
			DeckId: atraxa, Picks: []*thothv1.ReturnPick{{CollectionCardId: cobEN}, {CollectionCardId: cobJA}},
		}))
		if here := ownership(atraxa)[""].CheckedOutHere; here != 0 {
			t.Errorf("checked out after full return = %d, want 0", here)
		}
		must[*thothv1.DeleteDeckResponse](t)(decks.DeleteDeck(as(alice), &thothv1.DeleteDeckRequest{Id: atraxa}))
	})
}
