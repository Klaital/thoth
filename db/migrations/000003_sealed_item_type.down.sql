ALTER TABLE sealed_items
  DROP COLUMN item_type,
  MODIFY COLUMN set_code VARCHAR(16) NOT NULL DEFAULT '',
  CHANGE COLUMN name description TEXT NULL;
