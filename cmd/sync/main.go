package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"
	"path/filepath"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/klaital/thoth/internal/scryfall"
)

func main() {
	_ = godotenv.Load()

	defaultCache := filepath.Join(mustUserCacheDir(), "thoth", "scryfall")
	cacheDir := flag.String("cache-dir", defaultCache, "directory for cached Scryfall bulk data")
	flag.Parse()

	dsn := mustEnv("DB_DSN")

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("opening DB: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("connecting to DB: %v", err)
	}

	syncer := scryfall.NewSyncer(db, *cacheDir)
	stats, err := syncer.Sync(context.Background())
	if err != nil {
		log.Fatalf("sync failed: %v", err)
	}

	action := "used cached file"
	if stats.Downloaded {
		action = "downloaded fresh file"
	}
	log.Printf("sync complete: %d cards processed (%s)", stats.CardsProcessed, action)
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s is required", key)
	}
	return v
}

func mustUserCacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return os.TempDir()
	}
	return dir
}
