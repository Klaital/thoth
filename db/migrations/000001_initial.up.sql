-- Users
CREATE TABLE IF NOT EXISTS users (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    email         VARCHAR(255)    NOT NULL,
    password_hash VARCHAR(255)        NULL,
    google_id     VARCHAR(255)        NULL,
    created_at    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_users_email (email),
    UNIQUE KEY uq_users_google_id (google_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Collections
CREATE TABLE IF NOT EXISTS collections (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    owner_id    BIGINT UNSIGNED NOT NULL,
    name        VARCHAR(255)    NOT NULL,
    description TEXT                NULL,
    type        ENUM('mtg_cards','mtg_sealed','mtg_deck','books','dvds') NOT NULL,
    is_public   TINYINT(1)      NOT NULL DEFAULT 0,
    created_at  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    CONSTRAINT fk_collections_owner FOREIGN KEY (owner_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Scryfall card catalog (synced from Scryfall bulk data)
CREATE TABLE IF NOT EXISTS scryfall_cards (
    id               VARCHAR(36)  NOT NULL,
    name             VARCHAR(255) NOT NULL,
    set_code         VARCHAR(16)  NOT NULL,
    collector_number VARCHAR(16)  NOT NULL,
    image_uri        TEXT             NULL,
    prices_json      JSON             NULL,
    synced_at        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_scryfall_cards_name (name),
    KEY idx_scryfall_cards_set (set_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Card items (for mtg_cards and mtg_deck collections)
CREATE TABLE IF NOT EXISTS card_items (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    collection_id    BIGINT UNSIGNED NOT NULL,
    scryfall_card_id VARCHAR(36)     NOT NULL,
    foil_treatment   ENUM('nonfoil','foil','etched') NOT NULL DEFAULT 'nonfoil',
    quantity         INT             NOT NULL DEFAULT 1,
    notes            TEXT                NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_card_items (collection_id, scryfall_card_id, foil_treatment),
    CONSTRAINT fk_card_items_collection FOREIGN KEY (collection_id) REFERENCES collections (id) ON DELETE CASCADE,
    CONSTRAINT fk_card_items_scryfall   FOREIGN KEY (scryfall_card_id) REFERENCES scryfall_cards (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Sealed product items (for mtg_sealed collections)
CREATE TABLE IF NOT EXISTS sealed_items (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    collection_id BIGINT UNSIGNED NOT NULL,
    set_code      VARCHAR(16)     NOT NULL,
    description   TEXT                NULL,
    quantity      INT             NOT NULL DEFAULT 1,
    notes         TEXT                NULL,
    PRIMARY KEY (id),
    CONSTRAINT fk_sealed_items_collection FOREIGN KEY (collection_id) REFERENCES collections (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Book/DVD items (for books and dvds collections)
CREATE TABLE IF NOT EXISTS book_dvd_items (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    collection_id BIGINT UNSIGNED NOT NULL,
    upc           VARCHAR(32)     NOT NULL,
    title         VARCHAR(255)    NOT NULL,
    quantity      INT             NOT NULL DEFAULT 1,
    notes         TEXT                NULL,
    PRIMARY KEY (id),
    CONSTRAINT fk_book_dvd_items_collection FOREIGN KEY (collection_id) REFERENCES collections (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
