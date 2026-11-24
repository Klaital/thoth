-- name: GetUsageTotals :one
-- Totals for the usage gauges exported by internal/telemetry. Polled once per
-- metric export interval (60s); fine at current table sizes.
SELECT
    (SELECT COUNT(*) FROM users)                           AS users,
    (SELECT COUNT(*) FROM collections)                     AS collections,
    (SELECT COUNT(*) FROM decks)                           AS decks,
    CAST((SELECT COALESCE(SUM(quantity), 0) FROM collection_cards) AS SIGNED) AS cards;
