-- Owned cards: exact printings stored in a collection (table collection_cards).
-- Query names keep the CardItem prefix used by the API.

-- name: GetCardItemByID :one
SELECT ci.*, sc.name AS card_name, sc.set_code, sc.collector_number, sc.image_uri, sc.scryfall_uri, sc.prices_json
FROM collection_cards ci
JOIN scryfall_cards sc ON sc.id = ci.scryfall_card_id
WHERE ci.id = ? LIMIT 1;

-- name: ListCardItemsByCollection :many
SELECT ci.*, sc.name AS card_name, sc.set_code, sc.collector_number, sc.image_uri, sc.scryfall_uri, sc.prices_json
FROM collection_cards ci
JOIN scryfall_cards sc ON sc.id = ci.scryfall_card_id
WHERE ci.collection_id = ?
ORDER BY sc.name;

-- name: UpsertCardItem :execresult
INSERT INTO collection_cards (collection_id, scryfall_card_id, foil_treatment, language, quantity, notes)
VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  quantity = quantity + VALUES(quantity),
  notes = VALUES(notes);

-- name: UpdateCardItem :exec
UPDATE collection_cards SET quantity = ?, notes = ?, language = ?
WHERE id = ?;

-- name: FindCardItem :one
SELECT * FROM collection_cards
WHERE collection_id = ? AND scryfall_card_id = ? AND foil_treatment = ? AND language = ?
LIMIT 1;

-- name: DeleteCardItem :exec
DELETE FROM collection_cards WHERE id = ?;

-- name: ListCardItemsByIDs :many
SELECT * FROM collection_cards
WHERE collection_id = ? AND id IN (sqlc.slice('ids'));

-- name: MoveCardItem :exec
UPDATE collection_cards SET collection_id = ? WHERE id = ?;

-- name: DeleteCardItemsByIDs :exec
DELETE FROM collection_cards
WHERE collection_id = ? AND id IN (sqlc.slice('ids'));

-- name: GetCardItemByKey :one
SELECT ci.*, sc.name AS card_name, sc.set_code, sc.collector_number, sc.image_uri, sc.scryfall_uri, sc.prices_json
FROM collection_cards ci
JOIN scryfall_cards sc ON sc.id = ci.scryfall_card_id
WHERE ci.collection_id = ? AND ci.scryfall_card_id = ? AND ci.foil_treatment = ? AND ci.language = ?
LIMIT 1;

-- name: SearchCardItemsByOwner :many
SELECT ci.*, sc.name AS card_name, sc.set_code, sc.collector_number, sc.image_uri, sc.scryfall_uri, sc.prices_json
FROM collection_cards ci
JOIN scryfall_cards sc ON sc.id = ci.scryfall_card_id
JOIN collections c ON c.id = ci.collection_id
WHERE c.owner_id = ? AND sc.name LIKE CONCAT('%', sqlc.arg('name_query'), '%')
ORDER BY ci.collection_id, sc.name
LIMIT 1000;
