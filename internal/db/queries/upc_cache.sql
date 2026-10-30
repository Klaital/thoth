-- name: GetUPCCache :one
SELECT * FROM upc_cache WHERE upc = ? LIMIT 1;

-- name: UpsertUPCCache :exec
INSERT INTO upc_cache (upc, raw_name, raw_description, parsed_name, parsed_set_name, set_code, item_type)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  raw_name        = VALUES(raw_name),
  raw_description = VALUES(raw_description),
  parsed_name     = VALUES(parsed_name),
  parsed_set_name = VALUES(parsed_set_name),
  set_code        = VALUES(set_code),
  item_type       = VALUES(item_type),
  fetched_at      = CURRENT_TIMESTAMP;
