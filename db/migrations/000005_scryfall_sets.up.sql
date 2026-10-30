CREATE TABLE IF NOT EXISTS scryfall_sets (
    code         VARCHAR(16)  NOT NULL,
    name         VARCHAR(255) NOT NULL,
    released_at  DATE             NULL,
    icon_svg_uri TEXT             NULL,
    synced_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (code),
    KEY idx_scryfall_sets_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
