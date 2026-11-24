-- Best-effort reverse of 000009. The intended rollback is restoring the dump
-- taken before the up migration; this exists so `migrate down` works in dev.
-- Lossy: card languages are merged away, deck cards without a printing get an
-- arbitrary printing of that name (or are dropped if the name is unknown), and
-- default collections become ordinary collections.

CREATE TABLE card_items (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    collection_id    BIGINT UNSIGNED NOT NULL,
    scryfall_card_id VARCHAR(36)     NOT NULL,
    foil_treatment   ENUM('nonfoil','foil','etched') NOT NULL DEFAULT 'nonfoil',
    category         ENUM('none','commander','deck','sideboard') NOT NULL DEFAULT 'none',
    quantity         INT             NOT NULL DEFAULT 1,
    notes            TEXT                NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_card_items_category (collection_id, scryfall_card_id, foil_treatment, category),
    CONSTRAINT fk_card_items_collection FOREIGN KEY (collection_id) REFERENCES collections (id) ON DELETE CASCADE,
    CONSTRAINT fk_card_items_scryfall   FOREIGN KEY (scryfall_card_id) REFERENCES scryfall_cards (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE collections
    DROP KEY uq_collections_default,
    DROP COLUMN default_owner_key,
    DROP COLUMN is_default,
    MODIFY type ENUM('mtg_cards','mtg_sealed','mtg_deck','books','dvds') NOT NULL,
    ADD COLUMN legacy_deck_id BIGINT UNSIGNED NULL;

INSERT INTO card_items (collection_id, scryfall_card_id, foil_treatment, category, quantity, notes)
SELECT collection_id, scryfall_card_id, foil_treatment, 'none', SUM(quantity),
       GROUP_CONCAT(NULLIF(TRIM(notes), '') SEPARATOR '\n')
FROM collection_cards
GROUP BY collection_id, scryfall_card_id, foil_treatment;

INSERT INTO collections (owner_id, name, description, type, is_public, created_at, updated_at, legacy_deck_id)
SELECT owner_id, name, notes, 'mtg_deck', is_public, created_at, updated_at, id
FROM decks;

INSERT INTO card_items (collection_id, scryfall_card_id, foil_treatment, category, quantity)
SELECT c.id,
       COALESCE(dc.preferred_scryfall_card_id,
                (SELECT MIN(sc.id) FROM scryfall_cards sc WHERE sc.name = dc.card_name)) AS printing,
       'nonfoil',
       CASE dc.board WHEN 'commander' THEN 'commander'
                     WHEN 'sideboard' THEN 'sideboard'
                     ELSE 'deck' END,
       dc.quantity
FROM deck_cards dc
JOIN collections c ON c.legacy_deck_id = dc.deck_id
HAVING printing IS NOT NULL
ON DUPLICATE KEY UPDATE quantity = card_items.quantity + VALUES(quantity);

ALTER TABLE collections DROP COLUMN legacy_deck_id;

DROP TABLE collection_cards;
DROP TABLE deck_cards;
DROP TABLE decks;
