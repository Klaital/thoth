-- Fails if the same card/foil exists in more than one category; merge those rows first.
ALTER TABLE card_items
  ADD UNIQUE KEY uq_card_items (collection_id, scryfall_card_id, foil_treatment),
  DROP KEY uq_card_items_category,
  DROP COLUMN category;
