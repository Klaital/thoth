-- name: GetSealedItemByID :one
SELECT * FROM sealed_items WHERE id = ? LIMIT 1;

-- name: ListSealedItemsByCollection :many
SELECT * FROM sealed_items WHERE collection_id = ? ORDER BY set_code;

-- name: CreateSealedItem :execresult
INSERT INTO sealed_items (collection_id, set_code, description, quantity, notes)
VALUES (?, ?, ?, ?, ?);

-- name: UpdateSealedItem :exec
UPDATE sealed_items SET description = ?, quantity = ?, notes = ?
WHERE id = ?;

-- name: DeleteSealedItem :exec
DELETE FROM sealed_items WHERE id = ?;
