-- name: GetUserByID :one
SELECT * FROM users WHERE id = ? LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ? LIMIT 1;

-- name: GetUserByGoogleID :one
SELECT * FROM users WHERE google_id = ? LIMIT 1;

-- name: CreateUser :execresult
INSERT INTO users (email, password_hash, google_id)
VALUES (?, ?, ?);

-- name: UpsertGoogleUser :execresult
INSERT INTO users (email, google_id)
VALUES (?, ?)
ON DUPLICATE KEY UPDATE google_id = VALUES(google_id), updated_at = CURRENT_TIMESTAMP;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;
