package scryfall

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const bulkDataURL = "https://api.scryfall.com/bulk-data"

type bulkDataEntry struct {
	Type        string `json:"type"`
	DownloadURI string `json:"download_uri"`
}

type bulkDataResponse struct {
	Data []bulkDataEntry `json:"data"`
}

// Card is a minimal subset of a Scryfall card object used for sync.
type Card struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Set             string            `json:"set"`
	CollectorNumber string            `json:"collector_number"`
	ImageURIs       map[string]string `json:"image_uris"`
	Prices          map[string]string `json:"prices"`
}

func fetchDefaultCardsURL(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bulkDataURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "ThothsCollection/0.1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching bulk data index: %w", err)
	}
	defer resp.Body.Close()

	var bdr bulkDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&bdr); err != nil {
		return "", err
	}

	for _, entry := range bdr.Data {
		if entry.Type == "default_cards" {
			return entry.DownloadURI, nil
		}
	}
	return "", fmt.Errorf("default_cards bulk data entry not found")
}

// StreamCards fetches the Scryfall default_cards bulk file and calls fn for each card.
// The file is ~200MB; streaming avoids loading it all into memory.
func StreamCards(ctx context.Context, fn func(Card) error) error {
	downloadURL, err := fetchDefaultCardsURL(ctx)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "ThothsCollection/0.1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading bulk data: %w", err)
	}
	defer resp.Body.Close()

	return streamDecodeCards(resp.Body, fn)
}

func streamDecodeCards(r io.Reader, fn func(Card) error) error {
	dec := json.NewDecoder(r)

	// Read opening '['
	if _, err := dec.Token(); err != nil {
		return fmt.Errorf("reading opening bracket: %w", err)
	}

	for dec.More() {
		var card Card
		if err := dec.Decode(&card); err != nil {
			return fmt.Errorf("decoding card: %w", err)
		}
		if err := fn(card); err != nil {
			return err
		}
	}
	return nil
}
