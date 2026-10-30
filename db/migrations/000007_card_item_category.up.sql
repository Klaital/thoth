-- The new unique key is added before the old one is dropped because the
-- collection_id foreign key needs an index that starts with collection_id.
ALTER TABLE card_items
  ADD COLUMN category ENUM('none','commander','deck','sideboard') NOT NULL DEFAULT 'none' AFTER foil_treatment,
  ADD UNIQUE KEY uq_card_items_category (collection_id, scryfall_card_id, foil_treatment, category),
  DROP KEY uq_card_items;
