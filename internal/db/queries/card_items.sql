-- name: GetCardItemByID :one
SELECT ci.*, sc.name AS card_name, sc.set_code, sc.collector_number, sc.image_uri, sc.prices_json
FROM card_items ci
JOIN scryfall_cards sc ON sc.id = ci.scryfall_card_id
WHERE ci.id = ? LIMIT 1;

-- name: ListCardItemsByCollection :many
SELECT ci.*, sc.name AS card_name, sc.set_code, sc.collector_number, sc.image_uri, sc.prices_json
FROM card_items ci
JOIN scryfall_cards sc ON sc.id = ci.scryfall_card_id
WHERE ci.collection_id = ?
ORDER BY sc.name;

-- name: UpsertCardItem :execresult
INSERT INTO card_items (collection_id, scryfall_card_id, foil_treatment, category, quantity, notes)
VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  quantity = quantity + VALUES(quantity),
  notes = VALUES(notes);

-- name: UpdateCardItem :exec
UPDATE card_items SET quantity = ?, notes = ?, category = ?
WHERE id = ?;

-- name: FindCardItem :one
SELECT * FROM card_items
WHERE collection_id = ? AND scryfall_card_id = ? AND foil_treatment = ? AND category = ?
LIMIT 1;

-- name: DeleteCardItem :exec
DELETE FROM card_items WHERE id = ?;

-- name: ListCommandersByCollections :many
SELECT ci.collection_id, sc.name AS card_name, sc.image_uri
FROM card_items ci
JOIN scryfall_cards sc ON sc.id = ci.scryfall_card_id
WHERE ci.category = 'commander' AND ci.collection_id IN (sqlc.slice('collection_ids'))
ORDER BY ci.collection_id, sc.name;

-- name: ListCardItemsByIDs :many
SELECT * FROM card_items
WHERE collection_id = ? AND id IN (sqlc.slice('ids'));

-- name: MoveCardItem :exec
UPDATE card_items SET collection_id = ? WHERE id = ?;

-- name: DeleteCardItemsByIDs :exec
DELETE FROM card_items
WHERE collection_id = ? AND id IN (sqlc.slice('ids'));
