package scryfall

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	bulkDataURL   = "https://api.scryfall.com/bulk-data/default_cards"
	cacheFile     = "default_cards.jsonl.gz"
	cacheMetaFile = "default_cards.jsonl.gz.meta"

	// Scryfall asks clients not to re-fetch bulk data more than once per day.
	minFetchInterval = 24 * time.Hour
)

type cacheMeta struct {
	UpdatedAt time.Time `json:"updated_at"`
	FetchedAt time.Time `json:"fetched_at"`
}

// FetchBulkDataInfo retrieves the current metadata for the default_cards bulk file.
func FetchBulkDataInfo() (BulkDataInfo, error) {
	req, err := http.NewRequest(http.MethodGet, bulkDataURL, nil)
	if err != nil {
		return BulkDataInfo{}, err
	}
	req.Header.Set("User-Agent", "thoth-collection-manager/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return BulkDataInfo{}, fmt.Errorf("fetching bulk data info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return BulkDataInfo{}, fmt.Errorf("bulk data info: HTTP %d", resp.StatusCode)
	}

	var info BulkDataInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return BulkDataInfo{}, fmt.Errorf("decoding bulk data info: %w", err)
	}
	return info, nil
}

// EnsureCached downloads the bulk data file only if:
//   - the cache file does not exist, OR
//   - the Scryfall file was updated after our cached version
//
// The download is written to a temp file first and atomically renamed to avoid
// leaving a partial file on disk if the process is interrupted.
// Returns true if a new file was downloaded.
func EnsureCached(info BulkDataInfo, cacheDir string) (bool, error) {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return false, fmt.Errorf("creating cache dir: %w", err)
	}

	metaPath := filepath.Join(cacheDir, cacheMetaFile)
	cardPath := filepath.Join(cacheDir, cacheFile)

	// Check whether we already have an up-to-date copy.
	if meta, err := readMeta(metaPath); err == nil {
		_, statErr := os.Stat(cardPath)
		if statErr == nil && !info.UpdatedAt.After(meta.UpdatedAt) {
			return false, nil // cache is current
		}
		// Respect the minimum fetch interval even if the remote claims an update.
		if statErr == nil && time.Since(meta.FetchedAt) < minFetchInterval {
			return false, nil
		}
	}

	if err := download(info.DownloadURI, cardPath); err != nil {
		return false, err
	}

	meta := cacheMeta{UpdatedAt: info.UpdatedAt, FetchedAt: time.Now().UTC()}
	if err := writeMeta(metaPath, meta); err != nil {
		return true, fmt.Errorf("writing cache meta: %w", err)
	}
	return true, nil
}

// CacheFilePath returns the path to the cached bulk data JSON file.
func CacheFilePath(cacheDir string) string {
	return filepath.Join(cacheDir, cacheFile)
}

// noDecompressClient is a one-off HTTP client that never transparently inflates
// gzip responses. We want the raw compressed bytes written to disk.
var noDecompressClient = &http.Client{
	Transport: &http.Transport{DisableCompression: true},
}

func download(uri, destPath string) error {
	if uri == "" {
		return fmt.Errorf("no download URI provided")
	}
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "thoth-collection-manager/1.0")

	resp, err := noDecompressClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading bulk data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading bulk data: HTTP %d", resp.StatusCode)
	}

	// Write to a temp file in the same directory, then rename for atomicity.
	tmp, err := os.CreateTemp(filepath.Dir(destPath), "scryfall-*.json.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpName) // no-op if rename succeeded
	}()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return fmt.Errorf("writing bulk data: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(tmpName, destPath); err != nil {
		return fmt.Errorf("moving bulk data into place: %w", err)
	}
	return nil
}

func readMeta(path string) (cacheMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return cacheMeta{}, err
	}
	defer f.Close()
	var m cacheMeta
	return m, json.NewDecoder(f).Decode(&m)
}

func writeMeta(path string, m cacheMeta) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(m)
}
