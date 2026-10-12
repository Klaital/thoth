package scryfall

import "time"

// BulkDataInfo is the metadata returned by GET /bulk-data/default_cards.
type BulkDataInfo struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	UpdatedAt   time.Time `json:"updated_at"`
	DownloadURI string    `json:"jsonl_download_uri"`
	Size        int64     `json:"compressed_size"`
}

type imageURIs struct {
	Normal string `json:"normal"`
	Small  string `json:"small"`
}

type cardFace struct {
	ImageURIs *imageURIs `json:"image_uris"`
}

// RawCard is the minimal subset of fields extracted from the Scryfall bulk data file.
type RawCard struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Set             string         `json:"set"`
	CollectorNumber string         `json:"collector_number"`
	ImageURIs       *imageURIs     `json:"image_uris"`
	CardFaces       []cardFace     `json:"card_faces"`
	Prices          map[string]any `json:"prices"`
}

// BestImageURI returns the normal image URI, falling back to the first card face
// for double-faced cards where the top-level image_uris is absent.
func (c *RawCard) BestImageURI() string {
	if c.ImageURIs != nil && c.ImageURIs.Normal != "" {
		return c.ImageURIs.Normal
	}
	if len(c.CardFaces) > 0 && c.CardFaces[0].ImageURIs != nil {
		return c.CardFaces[0].ImageURIs.Normal
	}
	return ""
}

// SyncStats summarises a completed sync run.
type SyncStats struct {
	CardsProcessed int
	Downloaded     bool // true if a fresh file was fetched from Scryfall
}
