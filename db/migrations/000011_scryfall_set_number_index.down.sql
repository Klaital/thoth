ALTER TABLE scryfall_cards
    ADD KEY idx_scryfall_cards_set (set_code),
    DROP KEY idx_scryfall_cards_set_number;
