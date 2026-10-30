ALTER TABLE upc_cache
  ADD COLUMN set_code VARCHAR(16) NULL AFTER parsed_set_name;
