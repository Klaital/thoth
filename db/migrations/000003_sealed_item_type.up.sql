ALTER TABLE sealed_items
  ADD COLUMN item_type ENUM(
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
  ) NOT NULL DEFAULT 'play_booster_box' AFTER collection_id,
  MODIFY COLUMN set_code VARCHAR(16) NULL,
  CHANGE COLUMN description name TEXT NULL;
