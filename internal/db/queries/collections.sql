-- name: GetCollectionByID :one
SELECT * FROM collections WHERE id = ? LIMIT 1;

-- name: ListCollectionsByOwner :many
SELECT * FROM collections WHERE owner_id = ? ORDER BY created_at DESC;

-- name: ListPublicCollectionsByOwner :many
SELECT * FROM collections WHERE owner_id = ? AND is_public = 1 ORDER BY created_at DESC;

-- name: CreateCollection :execresult
INSERT INTO collections (owner_id, name, description, type, is_public)
VALUES (?, ?, ?, ?, ?);

-- name: UpdateCollection :exec
UPDATE collections
SET name = ?, description = ?, is_public = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeleteCollection :exec
DELETE FROM collections WHERE id = ?;

-- name: GetDefaultCollection :one
SELECT * FROM collections WHERE owner_id = ? AND is_default = 1 LIMIT 1;

-- name: CreateDefaultCollection :execresult
-- INSERT IGNORE: a concurrent request may have created it first (unique
-- uq_collections_default); the caller re-reads with GetDefaultCollection.
INSERT IGNORE INTO collections (owner_id, name, type, is_default)
VALUES (?, 'Default', 'mtg_cards', 1);
