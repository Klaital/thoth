-- Cards checked out of a collection into a deck (TH-17). A checkout holds
-- some of a collection_cards row's copies; the row itself stays put.

-- name: ListOwnedCopiesForDeck :many
-- Every copy the owner has of a card the deck lists, plus any copy already
-- checked out to the deck whose name is no longer in it.
SELECT ci.id, ci.collection_id, ci.scryfall_card_id, ci.foil_treatment, ci.language, ci.quantity,
       sc.name AS card_name, sc.set_code, sc.collector_number, sc.image_uri, sc.prices_json,
       c.name AS collection_name,
       CAST(COALESCE((SELECT SUM(x.quantity) FROM deck_checkouts x
                      WHERE x.collection_card_id = ci.id), 0) AS SIGNED) AS checked_out_total,
       CAST(COALESCE((SELECT SUM(x.quantity) FROM deck_checkouts x
                      WHERE x.collection_card_id = ci.id AND x.deck_id = sqlc.arg('deck_id')), 0) AS SIGNED) AS checked_out_here
FROM collection_cards ci
JOIN scryfall_cards sc ON sc.id = ci.scryfall_card_id
JOIN collections c ON c.id = ci.collection_id
WHERE c.owner_id = sqlc.arg('owner_id')
  AND c.type = 'mtg_cards'
  AND (sc.name IN (SELECT dc.card_name FROM deck_cards dc WHERE dc.deck_id = sqlc.arg('deck_id'))
       OR ci.id IN (SELECT x.collection_card_id FROM deck_checkouts x WHERE x.deck_id = sqlc.arg('deck_id')))
ORDER BY sc.name, c.name, ci.id;

-- name: ListCheckoutsByDeck :many
SELECT x.id, x.deck_id, x.collection_card_id, x.quantity, x.checked_out_at,
       ci.collection_id, c.name AS collection_name,
       ci.scryfall_card_id, ci.foil_treatment, ci.language,
       sc.name AS card_name, sc.set_code, sc.collector_number, sc.image_uri
FROM deck_checkouts x
JOIN collection_cards ci ON ci.id = x.collection_card_id
JOIN collections c ON c.id = ci.collection_id
JOIN scryfall_cards sc ON sc.id = ci.scryfall_card_id
WHERE x.deck_id = ?
ORDER BY c.name, c.id, sc.name, ci.id;

-- name: ListCheckoutsByCollectionCardIDs :many
SELECT x.collection_card_id, x.deck_id, d.name AS deck_name, x.quantity
FROM deck_checkouts x
JOIN decks d ON d.id = x.deck_id
WHERE x.collection_card_id IN (sqlc.slice('ids'))
ORDER BY x.collection_card_id, d.name, d.id;

-- name: LockCollectionCards :many
-- Locks the rows a check-out will draw from, so concurrent check-outs can't
-- both take the last free copy.
SELECT id FROM collection_cards WHERE id IN (sqlc.slice('ids')) FOR UPDATE;

-- name: GetCheckoutSources :many
SELECT ci.id, ci.quantity, c.owner_id, c.type AS collection_type, sc.name AS card_name,
       CAST(COALESCE((SELECT SUM(x.quantity) FROM deck_checkouts x
                      WHERE x.collection_card_id = ci.id), 0) AS SIGNED) AS checked_out_total
FROM collection_cards ci
JOIN collections c ON c.id = ci.collection_id
JOIN scryfall_cards sc ON sc.id = ci.scryfall_card_id
WHERE ci.id IN (sqlc.slice('ids'));

-- name: UpsertCheckout :exec
INSERT INTO deck_checkouts (deck_id, collection_card_id, quantity)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE quantity = quantity + VALUES(quantity);

-- name: GetCheckoutForUpdate :one
SELECT * FROM deck_checkouts
WHERE deck_id = ? AND collection_card_id = ?
FOR UPDATE;

-- name: SetCheckoutQuantity :exec
UPDATE deck_checkouts SET quantity = ? WHERE id = ?;

-- name: DeleteCheckout :exec
DELETE FROM deck_checkouts WHERE id = ?;

-- name: CountCheckoutsByDeck :one
SELECT COUNT(*) FROM deck_checkouts WHERE deck_id = ?;

-- name: SumCheckoutsForCard :one
SELECT CAST(COALESCE(SUM(quantity), 0) AS SIGNED) FROM deck_checkouts WHERE collection_card_id = ?;

-- name: MergeCheckoutsInto :exec
-- Before a collection card row is merged into another (move or language
-- change), carry its checkouts over to the surviving row.
INSERT INTO deck_checkouts (deck_id, collection_card_id, quantity, checked_out_at)
SELECT src.deck_id, sqlc.arg('to_id'), src.quantity, src.checked_out_at
FROM deck_checkouts src
WHERE src.collection_card_id = sqlc.arg('from_id')
ON DUPLICATE KEY UPDATE quantity = deck_checkouts.quantity + VALUES(quantity);
