-- Printing lookups filter on set and collector number together. The new index
-- also covers set-only lookups, so it replaces the set_code index.
ALTER TABLE scryfall_cards
    ADD KEY idx_scryfall_cards_set_number (set_code, collector_number),
    DROP KEY idx_scryfall_cards_set;
