-- TH-1: split decks out of collections, add card language and default collections.
--
-- MySQL DDL is not transactional: if this fails part-way the schema is left
-- half-migrated and golang-migrate marks the version dirty. Take a dump first
-- (make db-dump) and restore it rather than trying to repair by hand.

-- 1. Decks and deck cards. legacy_collection_id maps old mtg_deck collections
--    to their new deck rows and is dropped at the end.
CREATE TABLE decks (
    id                   BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    owner_id             BIGINT UNSIGNED NOT NULL,
    name                 VARCHAR(255)    NOT NULL,
    notes                TEXT                NULL,
    is_public            TINYINT(1)      NOT NULL DEFAULT 0,
    created_at           DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    legacy_collection_id BIGINT UNSIGNED     NULL,
    PRIMARY KEY (id),
    KEY idx_decks_owner (owner_id),
    CONSTRAINT fk_decks_owner FOREIGN KEY (owner_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Deck cards are identified by card name; the printing is only a preference.
CREATE TABLE deck_cards (
    id                         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    deck_id                    BIGINT UNSIGNED NOT NULL,
    card_name                  VARCHAR(255)    NOT NULL,
    preferred_scryfall_card_id VARCHAR(36)         NULL,
    board                      ENUM('commander','main','sideboard') NOT NULL DEFAULT 'main',
    quantity                   INT             NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE KEY uq_deck_cards (deck_id, card_name, board),
    CONSTRAINT fk_deck_cards_deck     FOREIGN KEY (deck_id) REFERENCES decks (id) ON DELETE CASCADE,
    CONSTRAINT fk_deck_cards_scryfall FOREIGN KEY (preferred_scryfall_card_id) REFERENCES scryfall_cards (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 2. Old deck collections become decks.
INSERT INTO decks (owner_id, name, notes, is_public, created_at, updated_at, legacy_collection_id)
SELECT owner_id, name, description, is_public, created_at, updated_at, id
FROM collections
WHERE type = 'mtg_deck';

-- 3. Their cards become deck cards, merged by name and board.
INSERT INTO deck_cards (deck_id, card_name, preferred_scryfall_card_id, board, quantity)
SELECT d.id,
       sc.name,
       MIN(ci.scryfall_card_id),
       CASE ci.category WHEN 'commander' THEN 'commander'
                        WHEN 'sideboard' THEN 'sideboard'
                        ELSE 'main' END AS board,
       SUM(ci.quantity)
FROM card_items ci
JOIN decks d           ON d.legacy_collection_id = ci.collection_id
JOIN scryfall_cards sc ON sc.id = ci.scryfall_card_id
GROUP BY d.id, sc.name, board;

-- 4. Owned cards move to collection_cards. Rows that differed only by
--    category are merged; their quantities are summed and notes joined.
CREATE TABLE collection_cards (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    collection_id    BIGINT UNSIGNED NOT NULL,
    scryfall_card_id VARCHAR(36)     NOT NULL,
    foil_treatment   ENUM('nonfoil','foil','etched') NOT NULL DEFAULT 'nonfoil',
    language         VARCHAR(8)      NOT NULL DEFAULT 'en',
    quantity         INT             NOT NULL DEFAULT 1,
    notes            TEXT                NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_collection_cards (collection_id, scryfall_card_id, foil_treatment, language),
    CONSTRAINT fk_collection_cards_collection FOREIGN KEY (collection_id) REFERENCES collections (id) ON DELETE CASCADE,
    CONSTRAINT fk_collection_cards_scryfall   FOREIGN KEY (scryfall_card_id) REFERENCES scryfall_cards (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO collection_cards (collection_id, scryfall_card_id, foil_treatment, language, quantity, notes)
SELECT ci.collection_id,
       ci.scryfall_card_id,
       ci.foil_treatment,
       'en',
       SUM(ci.quantity),
       GROUP_CONCAT(NULLIF(TRIM(ci.notes), '') SEPARATOR '\n')
FROM card_items ci
JOIN collections c ON c.id = ci.collection_id
WHERE c.type <> 'mtg_deck'
GROUP BY ci.collection_id, ci.scryfall_card_id, ci.foil_treatment;

-- 5. Remove the old deck collections (cascades to their card_items) and the
--    old table.
DELETE FROM collections WHERE type = 'mtg_deck';
DROP TABLE card_items;

-- 6. Collections: no more deck type; at most one default per owner, enforced
--    by a unique key on a column that is NULL for non-default collections.
--    It must be VIRTUAL: MySQL forbids STORED generated columns over a base
--    column (owner_id) whose foreign key cascades.
ALTER TABLE collections
    MODIFY type ENUM('mtg_cards','mtg_sealed','books','dvds') NOT NULL,
    ADD COLUMN is_default TINYINT(1) NOT NULL DEFAULT 0 AFTER is_public,
    ADD COLUMN default_owner_key BIGINT UNSIGNED AS (IF(is_default, owner_id, NULL)) VIRTUAL,
    ADD UNIQUE KEY uq_collections_default (default_owner_key);

-- 7. Every user gets a default card collection: their oldest existing card
--    collection if they have one, otherwise a new empty one.
UPDATE collections c
JOIN (SELECT MIN(id) AS id FROM collections WHERE type = 'mtg_cards' GROUP BY owner_id) oldest
  ON oldest.id = c.id
SET c.is_default = 1;

INSERT INTO collections (owner_id, name, type, is_default)
SELECT u.id, 'Default', 'mtg_cards', 1
FROM users u
WHERE NOT EXISTS (SELECT 1 FROM collections c WHERE c.owner_id = u.id AND c.is_default = 1);

-- 8. Done with the mapping column.
ALTER TABLE decks DROP COLUMN legacy_collection_id;
