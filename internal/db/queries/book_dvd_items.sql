-- name: GetBookDvdItemByID :one
SELECT * FROM book_dvd_items WHERE id = ? LIMIT 1;

-- name: ListBookDvdItemsByCollection :many
SELECT * FROM book_dvd_items WHERE collection_id = ? ORDER BY title;

-- name: CreateBookDvdItem :execresult
INSERT INTO book_dvd_items (collection_id, upc, title, quantity, notes)
VALUES (?, ?, ?, ?, ?);

-- name: UpdateBookDvdItem :exec
UPDATE book_dvd_items SET title = ?, quantity = ?, notes = ?
WHERE id = ?;

-- name: DeleteBookDvdItem :exec
DELETE FROM book_dvd_items WHERE id = ?;

-- name: MoveBookDvdItems :exec
UPDATE book_dvd_items SET collection_id = sqlc.arg('target_collection_id')
WHERE collection_id = sqlc.arg('source_collection_id') AND id IN (sqlc.slice('ids'));

-- name: DeleteBookDvdItemsByIDs :exec
DELETE FROM book_dvd_items
WHERE collection_id = ? AND id IN (sqlc.slice('ids'));
