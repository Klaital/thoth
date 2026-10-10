-- name: SearchScryfallCards :many
SELECT * FROM scryfall_cards
WHERE name LIKE CONCAT('%', ?, '%')
ORDER BY name
LIMIT 20;

-- name: GetScryfallCardByID :one
SELECT * FROM scryfall_cards WHERE id = ? LIMIT 1;

-- name: UpsertScryfallCard :exec
INSERT INTO scryfall_cards (id, name, set_code, collector_number, image_uri, prices_json, synced_at)
VALUES (?, ?, ?, ?, ?, ?, NOW())
ON DUPLICATE KEY UPDATE
  name             = VALUES(name),
  set_code         = VALUES(set_code),
  collector_number = VALUES(collector_number),
  image_uri        = VALUES(image_uri),
  prices_json      = VALUES(prices_json),
  synced_at        = VALUES(synced_at);
