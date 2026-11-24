package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Business/usage instruments. They are created from the global meter at init;
// the global provider delegates to whatever Setup installs later, and they are
// no-ops when telemetry is disabled.
//
// Keep attribute values to small fixed sets: never put user IDs, emails or
// other unbounded values on metrics (put them on spans instead).
var (
	meter = otel.Meter("github.com/klaital/thoth")

	Logins = must(meter.Int64Counter("thoth.auth.logins",
		metric.WithDescription("Login attempts by method and result"),
		metric.WithUnit("{login}")))
	Registrations = must(meter.Int64Counter("thoth.auth.registrations",
		metric.WithDescription("New user accounts created"),
		metric.WithUnit("{user}")))
	CollectionsCreated = must(meter.Int64Counter("thoth.collections.created",
		metric.WithUnit("{collection}")))
	DecksCreated = must(meter.Int64Counter("thoth.decks.created",
		metric.WithUnit("{deck}")))
	CardsAdded = must(meter.Int64Counter("thoth.cards.added",
		metric.WithDescription("Card copies added to collections, by source"),
		metric.WithUnit("{card}")))
	BulkImportLines = must(meter.Int64Counter("thoth.bulk_import.lines",
		metric.WithDescription("Bulk import lines processed, by kind and result"),
		metric.WithUnit("{line}")))
	DeckCheckouts = must(meter.Int64Counter("thoth.deck.checkouts",
		metric.WithDescription("Card copies checked out into decks"),
		metric.WithUnit("{card}")))
	DeckReturns = must(meter.Int64Counter("thoth.deck.returns",
		metric.WithDescription("Card copies returned from decks"),
		metric.WithUnit("{card}")))
	SyncCards = must(meter.Int64Counter("thoth.scryfall.sync.cards",
		metric.WithDescription("Cards processed by the Scryfall sync job"),
		metric.WithUnit("{card}")))
	SyncDuration = must(meter.Float64Histogram("thoth.scryfall.sync.duration",
		metric.WithDescription("Scryfall sync run duration"),
		metric.WithUnit("s")))
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// RecordLogin counts a login attempt. method is "password", "google" or
// "refresh"; a nil err counts as success.
func RecordLogin(ctx context.Context, method string, err error) {
	result := "ok"
	if err != nil {
		result = "fail"
	}
	Logins.Add(ctx, 1, metric.WithAttributes(
		attribute.String("method", method),
		attribute.String("result", result),
	))
}

// RecordBulkLine counts one bulk-import line. kind is "card", "deck" or
// "sealed"; ok reports whether the line was imported.
func RecordBulkLine(ctx context.Context, kind string, ok bool) {
	result := "ok"
	if !ok {
		result = "fail"
	}
	BulkImportLines.Add(ctx, 1, metric.WithAttributes(
		attribute.String("kind", kind),
		attribute.String("result", result),
	))
}

// Source attribute values for CardsAdded.
var (
	SourceSingle = metric.WithAttributes(attribute.String("source", "single"))
	SourceBulk   = metric.WithAttributes(attribute.String("source", "bulk_import"))
)

// UsageTotals is what the usage gauges report.
type UsageTotals struct {
	Users       int64
	Collections int64
	Decks       int64
	Cards       int64
}

// RegisterUsageGauges exports thoth.*.total gauges, computed by calling get
// once per metric export.
func RegisterUsageGauges(get func(context.Context) (UsageTotals, error)) error {
	users := must(meter.Int64ObservableGauge("thoth.users.total", metric.WithUnit("{user}")))
	collections := must(meter.Int64ObservableGauge("thoth.collections.total", metric.WithUnit("{collection}")))
	decks := must(meter.Int64ObservableGauge("thoth.decks.total", metric.WithUnit("{deck}")))
	cards := must(meter.Int64ObservableGauge("thoth.cards.total",
		metric.WithDescription("Card copies across all collections"),
		metric.WithUnit("{card}")))
	_, err := meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		t, err := get(ctx)
		if err != nil {
			return err
		}
		o.ObserveInt64(users, t.Users)
		o.ObserveInt64(collections, t.Collections)
		o.ObserveInt64(decks, t.Decks)
		o.ObserveInt64(cards, t.Cards)
		return nil
	}, users, collections, decks, cards)
	return err
}
