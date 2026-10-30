-- name: GetSealedItemByID :one
SELECT * FROM sealed_items WHERE id = ? LIMIT 1;

-- name: ListSealedItemsByCollection :many
SELECT * FROM sealed_items WHERE collection_id = ? ORDER BY item_type, set_code;

-- name: CreateSealedItem :execresult
INSERT INTO sealed_items (collection_id, item_type, set_code, name, quantity, notes)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateSealedItem :exec
UPDATE sealed_items SET item_type = ?, set_code = ?, name = ?, quantity = ?, notes = ?
WHERE id = ?;

-- name: DeleteSealedItem :exec
DELETE FROM sealed_items WHERE id = ?;

-- name: MoveSealedItems :exec
UPDATE sealed_items SET collection_id = sqlc.arg('target_collection_id')
WHERE collection_id = sqlc.arg('source_collection_id') AND id IN (sqlc.slice('ids'));

-- name: DeleteSealedItemsByIDs :exec
DELETE FROM sealed_items
WHERE collection_id = ? AND id IN (sqlc.slice('ids'));
