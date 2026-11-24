-- name: GetDeckByID :one
SELECT * FROM decks WHERE id = ? LIMIT 1;

-- name: ListDecksByOwner :many
SELECT * FROM decks WHERE owner_id = ? ORDER BY updated_at DESC;

-- name: ListPublicDecksByOwner :many
SELECT * FROM decks WHERE owner_id = ? AND is_public = 1 ORDER BY updated_at DESC;

-- name: CreateDeck :execresult
INSERT INTO decks (owner_id, name, notes, is_public)
VALUES (?, ?, ?, ?);

-- name: UpdateDeck :exec
UPDATE decks
SET name = ?, notes = ?, is_public = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: TouchDeck :exec
UPDATE decks SET updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: DeleteDeck :exec
DELETE FROM decks WHERE id = ?;
