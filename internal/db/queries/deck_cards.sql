-- Deck cards are identified by name. When no printing is preferred, the image
-- comes from an arbitrary printing with that name.

-- name: ListDeckCards :many
SELECT dc.*,
       COALESCE(pc.image_uri,    (SELECT sc.image_uri    FROM scryfall_cards sc WHERE sc.name = dc.card_name ORDER BY sc.id LIMIT 1)) AS image_uri,
       COALESCE(pc.scryfall_uri, (SELECT sc.scryfall_uri FROM scryfall_cards sc WHERE sc.name = dc.card_name ORDER BY sc.id LIMIT 1)) AS scryfall_uri,
       pc.set_code AS preferred_set_code,
       pc.collector_number AS preferred_collector_number
FROM deck_cards dc
LEFT JOIN scryfall_cards pc ON pc.id = dc.preferred_scryfall_card_id
WHERE dc.deck_id = ?
ORDER BY dc.board, dc.card_name;

-- name: GetDeckCardByID :one
SELECT * FROM deck_cards WHERE id = ? LIMIT 1;

-- name: GetDeckCardByKey :one
SELECT * FROM deck_cards WHERE deck_id = ? AND card_name = ? AND board = ? LIMIT 1;

-- name: UpsertDeckCard :execresult
INSERT INTO deck_cards (deck_id, card_name, preferred_scryfall_card_id, board, quantity)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  quantity = quantity + VALUES(quantity),
  preferred_scryfall_card_id = COALESCE(VALUES(preferred_scryfall_card_id), preferred_scryfall_card_id);

-- name: UpdateDeckCard :exec
UPDATE deck_cards SET quantity = ?, board = ?, preferred_scryfall_card_id = ?
WHERE id = ?;

-- name: DeleteDeckCard :exec
DELETE FROM deck_cards WHERE id = ?;

-- name: ListDeckCommanders :many
SELECT dc.deck_id, dc.card_name,
       COALESCE(pc.image_uri, (SELECT sc.image_uri FROM scryfall_cards sc WHERE sc.name = dc.card_name ORDER BY sc.id LIMIT 1)) AS image_uri
FROM deck_cards dc
LEFT JOIN scryfall_cards pc ON pc.id = dc.preferred_scryfall_card_id
WHERE dc.board = 'commander' AND dc.deck_id IN (sqlc.slice('deck_ids'))
ORDER BY dc.deck_id, dc.card_name;

-- name: GetCanonicalCardName :one
-- Case-insensitive exact match; returns the name as Scryfall spells it.
SELECT name FROM scryfall_cards WHERE name = ? LIMIT 1;

-- name: GetCardNameByFrontFace :one
-- Double-faced cards are named "Front // Back"; decks often list only the front.
SELECT name FROM scryfall_cards WHERE name LIKE CONCAT(sqlc.arg('front_face'), ' // %') LIMIT 1;

-- name: GetDeckCardDetail :one
-- Same columns as ListDeckCards, for one card.
SELECT dc.*,
       COALESCE(pc.image_uri,    (SELECT sc.image_uri    FROM scryfall_cards sc WHERE sc.name = dc.card_name ORDER BY sc.id LIMIT 1)) AS image_uri,
       COALESCE(pc.scryfall_uri, (SELECT sc.scryfall_uri FROM scryfall_cards sc WHERE sc.name = dc.card_name ORDER BY sc.id LIMIT 1)) AS scryfall_uri,
       pc.set_code AS preferred_set_code,
       pc.collector_number AS preferred_collector_number
FROM deck_cards dc
LEFT JOIN scryfall_cards pc ON pc.id = dc.preferred_scryfall_card_id
WHERE dc.id = ?;

-- name: GetCanonicalCardNames :many
-- Batch form of GetCanonicalCardName: the names, as Scryfall spells them,
-- that case-insensitively equal one of the inputs.
SELECT DISTINCT name FROM scryfall_cards WHERE name IN (sqlc.slice('names'));
