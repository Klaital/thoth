-- TH-17: cards checked out of a collection into a deck.
--
-- collection_cards.quantity stays the number of copies owned; the copies still
-- in the collection are quantity minus the checked-out total. The card row
-- never leaves its collection, so a return always goes back where it came from.
CREATE TABLE deck_checkouts (
    id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    deck_id            BIGINT UNSIGNED NOT NULL,
    collection_card_id BIGINT UNSIGNED NOT NULL,
    quantity           INT             NOT NULL,
    checked_out_at     DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_deck_checkouts (deck_id, collection_card_id),
    KEY idx_deck_checkouts_card (collection_card_id),
    CONSTRAINT chk_deck_checkouts_quantity CHECK (quantity > 0),
    CONSTRAINT fk_deck_checkouts_deck FOREIGN KEY (deck_id) REFERENCES decks (id) ON DELETE CASCADE,
    CONSTRAINT fk_deck_checkouts_card FOREIGN KEY (collection_card_id) REFERENCES collection_cards (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
