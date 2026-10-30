-- name: UpsertScryfallSet :exec
INSERT INTO scryfall_sets (code, name, released_at, icon_svg_uri, synced_at)
VALUES (?, ?, ?, ?, NOW())
ON DUPLICATE KEY UPDATE
  name         = VALUES(name),
  released_at  = VALUES(released_at),
  icon_svg_uri = VALUES(icon_svg_uri),
  synced_at    = VALUES(synced_at);

-- name: LookupSetCodeByName :one
SELECT code FROM scryfall_sets WHERE name = ? LIMIT 1;

-- name: ListSets :many
SELECT code, name, released_at, icon_svg_uri FROM scryfall_sets ORDER BY released_at DESC, code;
