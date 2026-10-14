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
INSERT INTO card_items (collection_id, scryfall_card_id, foil_treatment, quantity, notes)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  quantity = quantity + VALUES(quantity),
  notes = VALUES(notes);

-- name: UpdateCardItem :exec
UPDATE card_items SET quantity = ?, notes = ?
WHERE id = ?;

-- name: DeleteCardItem :exec
DELETE FROM card_items WHERE id = ?;
