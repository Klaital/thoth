package scryfall

import (
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/klaital/thoth/internal/db"
)

const (
	batchSize   = 500
	logInterval = 10_000
)

// Syncer coordinates fetching Scryfall bulk data and writing it to the database.
type Syncer struct {
	db       *sql.DB
	queries  *db.Queries
	cacheDir string
}

// NewSyncer creates a Syncer using the given database connection and cache directory.
func NewSyncer(sqlDB *sql.DB, cacheDir string) *Syncer {
	return &Syncer{db: sqlDB, queries: db.New(sqlDB), cacheDir: cacheDir}
}

// SyncSets fetches all sets from the Scryfall /sets endpoint and upserts them.
func (s *Syncer) SyncSets(ctx context.Context) (int, error) {
	resp, err := FetchSets()
	if err != nil {
		return 0, fmt.Errorf("fetching sets: %w", err)
	}
	for _, set := range resp.Data {
		var releasedAt sql.NullTime
		if set.ReleasedAt != "" {
			t, err := time.Parse("2006-01-02", set.ReleasedAt)
			if err == nil {
				releasedAt = sql.NullTime{Time: t, Valid: true}
			}
		}
		if err := s.queries.UpsertScryfallSet(ctx, db.UpsertScryfallSetParams{
			Code:       set.Code,
			Name:       set.Name,
			ReleasedAt: releasedAt,
			IconSvgUri: sql.NullString{String: set.IconSVGURI, Valid: set.IconSVGURI != ""},
		}); err != nil {
			return 0, fmt.Errorf("upserting set %s: %w", set.Code, err)
		}
	}
	return len(resp.Data), nil
}

// Sync runs a full sync:
//  1. Fetches all sets from the Scryfall /sets endpoint
//  2. Fetches the Scryfall bulk-data metadata
//  3. Downloads the bulk file only if it has been updated since the last fetch
//  4. Streams-parses the JSON array and bulk-upserts into the database
func (s *Syncer) Sync(ctx context.Context) (SyncStats, error) {
	nSets, err := s.SyncSets(ctx)
	if err != nil {
		return SyncStats{}, fmt.Errorf("syncing sets: %w", err)
	}
	log.Printf("scryfall: synced %d sets", nSets)

	info, err := FetchBulkDataInfo()
	if err != nil {
		return SyncStats{}, fmt.Errorf("fetching bulk data info: %w", err)
	}
	log.Printf("scryfall: bulk data last updated %s", info.UpdatedAt.Format("2006-01-02 15:04:05 UTC"))
	log.Printf("scryfall: bulk data size: %d bytes at %s", info.Size, info.DownloadURI)
	downloaded, err := EnsureCached(info, s.cacheDir)
	if err != nil {
		return SyncStats{}, fmt.Errorf("ensuring cache: %w", err)
	}
	if downloaded {
		log.Printf("scryfall: downloaded fresh bulk data file")
	} else {
		log.Printf("scryfall: using cached bulk data file")
	}

	n, err := s.loadFromFile(ctx, CacheFilePath(s.cacheDir))
	if err != nil {
		return SyncStats{}, err
	}
	return SyncStats{CardsProcessed: n, Downloaded: downloaded}, nil
}

func (s *Syncer) loadFromFile(ctx context.Context, path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("opening cache file: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return 0, fmt.Errorf("opening gzip stream: %w", err)
	}
	defer gz.Close()

	dec := json.NewDecoder(gz)

	var (
		batch []RawCard
		total int
	)

	// JSONL: one JSON object per line; Decoder reads one value at a time until EOF.
	for {
		var card RawCard
		if err := dec.Decode(&card); err != nil {
			if err == io.EOF {
				break
			}
			return total, fmt.Errorf("decoding card at index %d: %w", total, err)
		}
		batch = append(batch, card)

		if len(batch) >= batchSize {
			if err := bulkUpsert(ctx, s.db, batch); err != nil {
				return total, fmt.Errorf("upserting batch at card %d: %w", total, err)
			}
			total += len(batch)
			batch = batch[:0]
			if total%logInterval == 0 {
				log.Printf("scryfall: %d cards processed…", total)
			}
		}
	}

	// Flush any remaining cards.
	if len(batch) > 0 {
		if err := bulkUpsert(ctx, s.db, batch); err != nil {
			return total, fmt.Errorf("upserting final batch: %w", err)
		}
		total += len(batch)
	}

	return total, nil
}

// bulkUpsert inserts or updates a batch of cards in a single SQL statement.
func bulkUpsert(ctx context.Context, db *sql.DB, batch []RawCard) error {
	if len(batch) == 0 {
		return nil
	}

	placeholders := make([]string, len(batch))
	args := make([]any, 0, len(batch)*8)

	for i, card := range batch {
		placeholders[i] = "(?,?,?,?,?,?,?,?,NOW())"
		pricesJSON, err := json.Marshal(card.Prices)
		if err != nil {
			pricesJSON = []byte("null")
		}
		imageURI := card.BestImageURI()
		var imageArg any
		if imageURI != "" {
			imageArg = imageURI
		}
		args = append(args,
			card.ID,
			card.Name,
			card.Set,
			card.CollectorNumber,
			imageArg,
			card.Foil,
			card.Nonfoil,
			string(pricesJSON),
		)
	}

	query := `INSERT INTO scryfall_cards (id, name, set_code, collector_number, image_uri, foil, nonfoil, prices_json, synced_at) VALUES ` +
		strings.Join(placeholders, ",") +
		` ON DUPLICATE KEY UPDATE
			name             = VALUES(name),
			set_code         = VALUES(set_code),
			collector_number = VALUES(collector_number),
			image_uri        = VALUES(image_uri),
			foil             = VALUES(foil),
			nonfoil          = VALUES(nonfoil),
			prices_json      = VALUES(prices_json),
			synced_at        = VALUES(synced_at)`

	_, err := db.ExecContext(ctx, query, args...)
	return err
}
