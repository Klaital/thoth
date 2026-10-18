-- name: SearchScryfallCards :many
SELECT * FROM scryfall_cards
WHERE name LIKE CONCAT('%', ?, '%')
ORDER BY name
LIMIT 20;

-- name: SearchScryfallCardsBySet :many
SELECT * FROM scryfall_cards
WHERE name LIKE CONCAT('%', ?, '%')
  AND set_code = ?
ORDER BY name
LIMIT 20;

-- name: GetScryfallCardByID :one
SELECT * FROM scryfall_cards WHERE id = ? LIMIT 1;

-- name: ListScryfallSetCodes :many
SELECT DISTINCT set_code FROM scryfall_cards ORDER BY set_code;

-- name: UpsertScryfallCard :exec
INSERT INTO scryfall_cards (id, name, set_code, collector_number, image_uri, foil, nonfoil, prices_json, synced_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW())
ON DUPLICATE KEY UPDATE
  name             = VALUES(name),
  set_code         = VALUES(set_code),
  collector_number = VALUES(collector_number),
  image_uri        = VALUES(image_uri),
  foil             = VALUES(foil),
  nonfoil          = VALUES(nonfoil),
  prices_json      = VALUES(prices_json),
  synced_at        = VALUES(synced_at);
