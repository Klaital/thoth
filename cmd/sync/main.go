package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/klaital/thoth/internal/db"
	"github.com/klaital/thoth/internal/scryfall"
	"github.com/klaital/thoth/internal/telemetry"
)

func main() {
	_ = godotenv.Load()

	defaultCache := filepath.Join(mustUserCacheDir(), "thoth", "scryfall")
	cacheDir := flag.String("cache-dir", defaultCache, "directory for cached Scryfall bulk data")
	flag.Parse()

	dsn := mustEnv("DB_DSN")
	ctx := context.Background()

	shutdownTelemetry, err := telemetry.Setup(ctx, "thoth-sync")
	if err != nil {
		log.Fatalf("setting up telemetry: %v", err)
	}
	// Flush before any exit, including failures (log.Fatalf skips defers).
	flush := func() {
		flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := shutdownTelemetry(flushCtx); err != nil {
			log.Printf("telemetry shutdown: %v", err)
		}
	}

	sqlDB, err := db.Connect(ctx, dsn)
	if err != nil {
		flush()
		log.Fatalf("connecting to DB: %v", err)
	}
	defer sqlDB.Close()

	start := time.Now()
	syncer := scryfall.NewSyncer(sqlDB, *cacheDir)
	stats, err := syncer.Sync(ctx)
	result := "ok"
	if err != nil {
		result = "fail"
	}
	telemetry.SyncDuration.Record(ctx, time.Since(start).Seconds(),
		metric.WithAttributes(attribute.String("result", result)))
	if err != nil {
		flush()
		log.Fatalf("sync failed: %v", err)
	}
	telemetry.SyncCards.Add(ctx, int64(stats.CardsProcessed))
	defer flush()

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
