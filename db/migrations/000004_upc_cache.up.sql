CREATE TABLE IF NOT EXISTS upc_cache (
    upc             VARCHAR(32)  NOT NULL,
    raw_name        TEXT             NULL,
    raw_description TEXT             NULL,
    parsed_name     VARCHAR(255)     NULL,
    parsed_set_name VARCHAR(255)     NULL,
    item_type       ENUM(
        'secret_lair',
        'play_booster',
        'play_booster_box',
        'collector_booster',
        'collector_booster_box',
        'bundle',
        'draft_night',
        'draft_booster',
        'draft_booster_box',
        'set_booster',
        'set_booster_box',
        'jumpstart_booster',
        'jumpstart_booster_box',
        'commander_precon',
        'prerelease_pack'
    ) NULL,
    fetched_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (upc)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
